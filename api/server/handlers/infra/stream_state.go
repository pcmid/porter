package infra

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/porter-dev/porter/api/server/handlers"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/api/server/shared/config"
	"github.com/porter-dev/porter/api/server/shared/websocket"
	"github.com/porter-dev/porter/api/types"
	"github.com/porter-dev/porter/internal/models"
	"github.com/porter-dev/porter/provisioner/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type InfraStreamStateHandler struct {
	handlers.PorterHandlerWriter
}

func NewInfraStreamStateHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *InfraStreamStateHandler {
	return &InfraStreamStateHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *InfraStreamStateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	safeRW := r.Context().Value(types.RequestCtxWebsocketKey).(*websocket.WebsocketSafeReadWriter)
	infra, _ := r.Context().Value(types.InfraScope).(*models.Infra)
	operation, _ := r.Context().Value(types.OperationScope).(*models.Operation)
	workspaceID := models.GetWorkspaceID(infra, operation)

	conn, err := grpc.Dial("localhost:8082", grpc.WithInsecure())

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	client := pb.NewProvisionerClient(conn)

	header := metadata.New(map[string]string{
		"workspace_id": workspaceID,
	})

	ctx := metadata.NewOutgoingContext(context.Background(), header)

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	stream, err := client.GetStateUpdate(ctx, &pb.Infra{
		ProjectId: int64(infra.ProjectID),
		Id:        int64(infra.ID),
		Suffix:    infra.Suffix,
	})

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	errorchan := make(chan error)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		wg.Wait()
		close(errorchan)
	}()

	go func() {
		defer wg.Done()

		for {
			if _, _, err := safeRW.ReadMessage(); err != nil {
				errorchan <- nil
				fmt.Println("STATE STREAMER closing websocket goroutine")
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		for {

			stateUpdate, err := stream.Recv()

			if err != nil {
				if err == io.EOF || errors.Is(ctx.Err(), context.Canceled) {
					errorchan <- nil
				} else {
					errorchan <- err
				}

				fmt.Println("STATE STREAMER closing grpc goroutine")

				return
			}

			safeRW.WriteJSONWithChannel(stateUpdate, errorchan)
		}
	}()

	for err = range errorchan {
		if err != nil {
			c.HandleAPIErrorNoWrite(w, r, apierrors.NewErrInternal(err))
		}

		// close the grpc stream: do not check for error case since the stream could already be
		// closed
		stream.CloseSend()

		// close the websocket stream: do not check for error case since the WS could already be
		// closed
		safeRW.Close()

		// cancel the context set for the grpc stream to ensure that Recv is unblocked
		cancel()
	}
}
