import { Steps } from "main/home/onboarding/types";
import React, { useContext, useEffect, useState } from "react";
import { integrationList } from "shared/common";

import loading from "assets/loading.gif";

import styled, { keyframes } from "styled-components";
import { capitalize, readableDate } from "shared/string_utils";
import {
  Infrastructure,
  KindMap,
  Operation,
  OperationStatus,
  OperationType,
  TFResourceState,
  TFState,
} from "shared/types";
import api from "shared/api";
import Placeholder from "./Placeholder";
import Loading from "./Loading";
import ExpandedOperation from "main/home/infrastructure/components/ExpandedOperation";
import { Context } from "shared/Context";
import { useWebsockets } from "shared/hooks/useWebsockets";
import Description from "./Description";
import Heading from "./form-components/Heading";
import PorterFormWrapper from "./porter-form/PorterFormWrapper";
import SaveButton from "./SaveButton";
import { ProgressPlugin } from "webpack";

type Props = {
  infras: Infrastructure[];
  project_id: number;
  auto_expanded?: boolean;
};

const nameMap: { [key: string]: string } = {
  eks: "Elastic Kubernetes Service (EKS)",
  ecr: "Elastic Container Registry (ECR)",
  doks: "DigitalOcean Kubernetes Service (DOKS)",
  docr: "DigitalOcean Container Registry (DOCR)",
  gke: "Google Kubernetes Engine (GKE)",
  gcr: "Google Container Registry (GCR)",
  rds: "Amazon Relational Database (RDS)",
};

const ProvisionerStatus: React.FC<Props> = ({
  infras,
  project_id,
  auto_expanded,
}) => {
  const renderV1Infra = (infra: Infrastructure) => {
    let errors: string[] = [];

    if (infra.status == "destroyed" || infra.status == "deleted") {
      errors.push("This infrastructure was destroyed.");
    }

    let error = null;

    if (errors.length > 0) {
      error = errors.map((error, index) => {
        return <ExpandedError key={index}>{error}</ExpandedError>;
      });
    }

    return (
      <StyledInfraObject key={infra.id}>
        <InfraHeader>
          <Flex>
            {integrationList[infra.kind] && (
              <Icon src={integrationList[infra.kind].icon} />
            )}
            {KindMap[infra.kind]?.provider_name}
          </Flex>
          <Timestamp>Started {readableDate(infra.created_at)}</Timestamp>
        </InfraHeader>
        <ErrorWrapper>{error}</ErrorWrapper>
      </StyledInfraObject>
    );
  };

  const renderV2Infra = (infra: Infrastructure) => {
    return (
      <InfraObject
        key={infra.id}
        project_id={project_id}
        infra={infra}
        is_expanded={auto_expanded}
      />
    );
  };

  const renderInfras = () => {
    return infras.map((infra) => {
      if (infra.api_version == "" || infra.api_version == "v1") {
        return renderV1Infra(infra);
      }

      return renderV2Infra(infra);
    });
  };

  return <StyledProvisionerStatus>{renderInfras()}</StyledProvisionerStatus>;
};

export default ProvisionerStatus;

type InfraObjectProps = {
  infra: Infrastructure;
  project_id: number;
  is_expanded: boolean;
};

const InfraObject: React.FC<InfraObjectProps> = ({
  infra,
  project_id,
  is_expanded,
}) => {
  const [isExpanded, setIsExpanded] = useState(is_expanded);
  const [isInProgress, setIsInProgress] = useState(
    infra.status == "creating" ||
      infra.status == "updating" ||
      infra.status == "deleting"
  );
  const [fullInfra, setFullInfra] = useState<Infrastructure>(null);
  const [infraState, setInfraState] = useState<TFState>(null);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if ((isExpanded || isInProgress) && !fullInfra) {
      setIsLoading(true);

      api
        .getInfraByID(
          "<token>",
          {},
          {
            project_id: project_id,
            infra_id: infra.id,
          }
        )
        .then(({ data }) => {
          setFullInfra(data);
          setIsLoading(false);
        })
        .catch((err) => {
          console.error(err);
        });
    }
  }, [infra, project_id, isExpanded, isInProgress]);

  useEffect(() => {
    if ((isExpanded || isInProgress) && !infraState) {
      setIsLoading(true);

      api
        .getInfraState(
          "<token>",
          {},
          {
            project_id: project_id,
            infra_id: infra.id,
          }
        )
        .then(({ data }) => {
          setInfraState(data);
          setIsLoading(false);
        })
        .catch((err) => {
          console.error(err);
        });
    }
  }, [infra, project_id, isExpanded, isInProgress]);

  const renderExpandedContentsCreated = () => {
    return <OperationDetails infra={fullInfra} />;
  };

  const renderExpandedContents = () => {
    if (!isExpanded) {
      return null;
    } else if (fullInfra) {
      return renderExpandedContentsCreated();
    }

    return (
      <ErrorWrapper>
        <Placeholder>
          <Loading />{" "}
        </Placeholder>
      </ErrorWrapper>
    );
  };

  const renderTimestampSection = () => {
    let timestampLabel = "Started";

    switch (infra.status) {
      case "created":
        timestampLabel = "Created at";
        break;
      case "creating":
        timestampLabel = "Started creating at";
        break;
      case "deleted":
        timestampLabel = "Deleted at";
        break;
      case "deleting":
        timestampLabel = "Started deleting at";
        break;
      case "updating":
        timestampLabel = "Started updating at";
        break;
    }

    return (
      <Timestamp>
        {timestampLabel} {readableDate(infra.updated_at)}
      </Timestamp>
    );
  };

  return (
    <StyledInfraObject key={infra.id}>
      <InfraHeader
        onClick={() =>
          setIsExpanded((val) => {
            setIsLoading(true);
            return !val;
          })
        }
      >
        <Flex>
          {integrationList[infra.kind] && (
            <Icon src={integrationList[infra.kind].icon} />
          )}
          {KindMap[infra.kind]?.provider_name}
        </Flex>
        <Flex>
          {renderTimestampSection()}
          <ExpandIconContainer>
            <i className="material-icons expand-icon">
              {isExpanded ? "expand_less" : "expand_more"}
            </i>
          </ExpandIconContainer>
        </Flex>
      </InfraHeader>
      {renderExpandedContents()}
    </StyledInfraObject>
  );
};

type OperationDetailsProps = {
  infra: Infrastructure;
};

const OperationDetails: React.FunctionComponent<OperationDetailsProps> = ({
  infra,
}) => {
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);
  const [operation, setOperation] = useState<Operation>(null);
  const [infraState, setInfraState] = useState<TFState>(null);
  const [infraStateInitialized, setInfraStateInitialized] = useState(false);
  const { currentProject, setCurrentError } = useContext(Context);
  const [erroredResources, setErroredResources] = useState<TFResourceState[]>(
    []
  );
  const [createdResources, setCreatedResources] = useState<TFResourceState[]>(
    []
  );
  const [plannedResources, setPlannedResources] = useState<TFResourceState[]>(
    []
  );

  const { newWebsocket, openWebsocket, closeWebsocket } = useWebsockets();

  const parseOperationWebsocketEvent = (evt: MessageEvent) => {
    let { status, resource_id, error } = JSON.parse(evt.data);

    // if the status and resource_id are defined, add this to the infra state
    if (status && resource_id) {
      setInfraState((curr) => {
        let currCopy: TFState = {
          last_updated: curr.last_updated,
          operation_id: curr.operation_id,
          status: curr.status,
          resources: { ...curr.resources },
        };

        if (currCopy.resources[resource_id] && status == "deleted") {
          delete currCopy.resources[resource_id];
        } else if (currCopy.resources[resource_id]) {
          currCopy.resources[resource_id].status = status;
          currCopy.resources[resource_id].error = error;
        } else {
          currCopy.resources[resource_id] = {
            id: resource_id,
            status: status,
            error: error,
          };
        }

        return currCopy;
      });
    }
  };

  const setupOperationWebsocket = (websocketID: string) => {
    let apiPath = `/api/projects/${currentProject.id}/infras/${infra.id}/operations/${infra.latest_operation.id}/state`;

    const wsConfig = {
      onopen: () => {
        console.log(`connected to websocket:`, websocketID);
      },
      onmessage: parseOperationWebsocketEvent,
      onclose: () => {
        console.log(`closing websocket:`, websocketID);
      },
      onerror: (err: ErrorEvent) => {
        console.log(err);
        closeWebsocket(websocketID);
      },
    };

    newWebsocket(websocketID, apiPath, wsConfig);
    openWebsocket(websocketID);
  };

  useEffect(() => {
    // if the latest_operation is in progress, open a websocket
    if (infraStateInitialized && infra.latest_operation.status === "starting") {
      const websocketID = infra.latest_operation.id;

      setupOperationWebsocket(websocketID);

      return () => {
        closeWebsocket(websocketID);
      };
    }
  }, [infraStateInitialized]);

  useEffect(() => {
    api
      .getInfraState(
        "<token>",
        {},
        {
          project_id: currentProject.id,
          infra_id: infra.id,
        }
      )
      .then(({ data }) => {
        setInfraState(data);
        setIsLoading(false);
        setInfraStateInitialized(true);
      })
      .catch((err) => {
        console.error(err);
        setInfraState({
          last_updated: "",
          operation_id: infra.latest_operation.id,
          status: "creating",
          resources: {},
        });
        setInfraStateInitialized(true);
      });
  }, [currentProject, infra]);

  useEffect(() => {
    api
      .getOperation(
        "<token>",
        {},
        {
          project_id: currentProject.id,
          infra_id: infra.id,
          operation_id: infra.latest_operation.id,
        }
      )
      .then(({ data }) => {
        setOperation(data);
        setIsLoading(false);
      })
      .catch((err) => {
        console.error(err);
        setHasError(true);
        setCurrentError(err.response?.data?.error);
        setIsLoading(false);
      });
  }, [currentProject, infra]);

  useEffect(() => {
    if (infraState && infraState.resources) {
      setErroredResources(
        Object.keys(infraState.resources)
          .map((key) => {
            if (
              infraState.resources[key].error &&
              infraState.resources[key].error != null
            ) {
              return infraState.resources[key];
            }

            return null;
          })
          .filter((val) => val)
      );

      setCreatedResources(
        Object.keys(infraState.resources)
          .map((key) => {
            if (infraState.resources[key].status == "created") {
              return infraState.resources[key];
            }

            return null;
          })
          .filter((val) => val)
      );

      setPlannedResources(
        Object.keys(infraState.resources)
          .map((key) => {
            if (infraState.resources[key].status == "planned_create") {
              return infraState.resources[key];
            }

            return null;
          })
          .filter((val) => val)
      );
    }
  }, [infraState]);

  if (isLoading || !infraState) {
    return (
      <Placeholder>
        <Loading />
      </Placeholder>
    );
  }

  if (hasError) {
    return <Placeholder>Error</Placeholder>;
  }

  const getOperationDescription = (
    type: OperationType,
    status: OperationStatus,
    time: string
  ): string => {
    switch (type) {
      case "retry_create":
      case "create":
        if (status == "starting") {
          return (
            "Status: infrastructure creation in progress, started at " +
            readableDate(time)
          );
        } else if (status == "completed") {
          return (
            "Status: infrastructure creation completed at " + readableDate(time)
          );
        } else if (status == "errored") {
          return "Status: this infrastructure encountered an error while creating.";
        }
      case "update":
        if (status == "starting") {
          return (
            "Status: infrastructure update in progress, started at " +
            readableDate(time)
          );
        } else if (status == "completed") {
          return (
            "Status: infrastructure update completed at " + readableDate(time)
          );
        } else if (status == "errored") {
          return "Status: this infrastructure encountered an error while updating.";
        }
      case "retry_delete":
      case "delete":
        if (status == "starting") {
          return (
            "Status: infrastructure deletion in progress, started at " +
            readableDate(time)
          );
        } else if (status == "completed") {
          return (
            "Status: infrastructure deletion completed at " + readableDate(time)
          );
        } else if (status == "errored") {
          return "Status: this infrastructure encountered an error while deleting.";
        }
    }
  };

  const renderLoadingBar = (
    completedResourceCount: number,
    plannedResourceCount: number
  ) => {
    let width = (100.0 * completedResourceCount) / plannedResourceCount;

    let operationKind = "Created";

    switch (infra.latest_operation.type) {
      case "retry_create":
      case "create":
        operationKind = "Created";
        break;
      case "update":
        operationKind = "Updated";
        break;
      case "retry_delete":
      case "delete":
        operationKind = "Deleted";
    }

    return (
      <StatusContainer>
        <LoadingBar>
          <LoadingFill status="loading" width={width + "%"} />
        </LoadingBar>
        <ResourceNumber>{`${completedResourceCount} / ${plannedResourceCount} ${operationKind}`}</ResourceNumber>
      </StatusContainer>
    );
  };

  const renderErrorSection = () => {
    if (erroredResources.length > 0) {
      return (
        <>
          <Description>
            Encountered the following errors while provisioning:
          </Description>
          <ErrorWrapper>
            {erroredResources.map((resource, index) => {
              return (
                <ExpandedError key={index}>{resource.error}</ExpandedError>
              );
            })}
          </ErrorWrapper>
        </>
      );
    }
  };

  return (
    <StyledCard>
      {renderLoadingBar(
        createdResources.length,
        createdResources.length +
          erroredResources.length +
          plannedResources.length
      )}
      <Description>
        {getOperationDescription(
          operation.type,
          operation.status,
          operation.last_updated
        )}
      </Description>
      {renderErrorSection()}
    </StyledCard>
  );
};

const StyledCard = styled.div`
  padding: 12px 20px;
  max-height: 300px;
  overflow-y: auto;
`;

const Flex = styled.div`
  display: flex;
  align-items: center;
`;

const Timestamp = styled.div`
  font-size: 13px;
  font-weight: 400;
  color: #ffffff55;
`;

const Icon = styled.img`
  height: 20px;
  margin-right: 10px;
`;

const ErrorWrapper = styled.div`
  margin-top: 20px;
  overflow-y: auto;
  user-select: text;
  padding: 0 15px;
`;

const ExpandedError = styled.div`
  background: #ffffff22;
  border-radius: 5px;
  padding: 15px;
  font-size: 13px;
  font-family: monospace;
  border: 1px solid #aaaabb;
  margin-bottom: 17px;
  padding-bottom: 17px;
`;

const StatusContainer = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
`;

const StatusText = styled.div`
  font-size: 13px;
  margin-left: 15px;
  color: #aaaabb;
  font-weight: 400;
`;

const ResourceNumber = styled.div`
  font-size: 12px;
  margin-left: 7px;
  min-width: 100px;
  text-align: right;
  color: #aaaabb;
`;

const movingGradient = keyframes`
  0% {
      background-position: left bottom;
  }

  100% {
      background-position: right bottom;
  }
`;

const StyledProvisionerStatus = styled.div`
  margin-top: 25px;
`;

const StyledInfraObject = styled.div`
  background: #ffffff1a;
  border: 1px solid #aaaabb;
  border-radius: 5px;
  margin-bottom: 10px;
  position: relative;
`;

const InfraHeader = styled.div`
  font-size: 13px;
  font-weight: 500;
  justify-content: space-between;
  padding: 15px;
  display: flex;
  align-items: center;
  cursor: pointer;
  height: 50px;

  :hover {
    background: #ffffff12;
  }

  .expand-icon {
    display: none;
    color: #ffffff55;
  }

  :hover .expand-icon {
    display: inline-block;
  }
`;

const LoadingBar = styled.div`
  background: #ffffff22;
  width: 100%;
  height: 8px;
  overflow: hidden;
  border-radius: 100px;
`;

const LoadingFill = styled.div<{ width: string; status: string }>`
  width: ${(props) => props.width};
  background: ${(props) =>
    props.status === "successful"
      ? "rgb(56, 168, 138)"
      : props.status === "error"
      ? "#fcba03"
      : "linear-gradient(to right, #8ce1ff, #616FEE)"};
  height: 100%;
  background-size: 250% 100%;
  animation: ${movingGradient} 2s infinite;
  animation-timing-function: ease-in-out;
  animation-direction: alternate;
`;

const ExpandIconContainer = styled.div`
  width: 30px;
  margin-left: 10px;
  padding-top: 2px;
`;
