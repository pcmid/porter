import React, {
  FormEvent,
  FormEventHandler,
  useContext,
  useState,
} from "react";
import { capitalize } from "shared/string_utils";
import styled from "styled-components";
import { Environment } from "../types";
import Options from "components/OptionsDropdown";
import { useRouting } from "shared/routing";
import api from "shared/api";
import { Context } from "shared/Context";
import Modal from "main/home/modals/Modal";
import InputRow from "components/form-components/InputRow";
import DynamicLink from "components/DynamicLink";

type Props = {
  environment: Environment;
  onDelete: (env: Environment) => void;
};

const EnvironmentCard = ({ environment, onDelete }: Props) => {
  const { currentCluster, currentProject, setCurrentError } = useContext(
    Context
  );
  const { pushFiltered } = useRouting();

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmationRepoName, setDeleteConfirmationRepoName] = useState(
    ""
  );

  const {
    id,
    name,
    deployment_count,
    git_repo_owner,
    git_repo_name,
    git_installation_id,
    last_deployment_status,
  } = environment;

  const showOpenPrs = () => {
    pushFiltered("/preview-environments", [], {
      current_tab: "pull_requests",
      repository: `${git_repo_owner}/${git_repo_name}`,
    });
  };

  const handleDelete = () => {
    if (!canDelete()) {
      return;
    }
    api
      .deleteEnvironment(
        "<token>",
        {
          name: name,
        },
        {
          project_id: currentProject.id,
          cluster_id: currentCluster.id,
          git_installation_id: git_installation_id,
          git_repo_owner: git_repo_owner,
          git_repo_name: git_repo_name,
        }
      )
      .then(() => {
        onDelete(environment);
        closeForm();
      })
      .catch((err) => {
        setCurrentError(JSON.stringify(err));
      });
  };

  const closeForm = () => {
    setShowDeleteModal(false);
    setDeleteConfirmationRepoName("");
  };

  const canDelete = () => {
    const repoName = deleteConfirmationRepoName;
    return repoName === `${git_repo_owner}/${git_repo_name}`;
  };

  return (
    <>
      {showDeleteModal ? (
        <Modal
          title={`Remove Preview Envs for ${git_repo_owner}/${git_repo_name}`}
          width="800px"
          height="260px"
          onRequestClose={closeForm}
        >
          <Warning highlight>
            ⚠️ All Preview Environment deployments associated with this repo will be deleted.
          </Warning>
          <InputRow
            type="text"
            label="Enter the full name of the repository to delete Preview Environments:"
            value={deleteConfirmationRepoName}
            placeholder={`${git_repo_owner}/${git_repo_name}`}
            setValue={(x: string) => setDeleteConfirmationRepoName(x)}
            width={"500px"}
          />
          <ActionWrapper>
            <DeleteButton
              onClick={() => handleDelete()}
              disabled={!canDelete()}
            >
              Delete
            </DeleteButton>
          </ActionWrapper>
        </Modal>
      ) : null}
      <EnvironmentCardWrapper onClick={showOpenPrs}>
        <DataContainer>
          <RepoName>
            <Icon
              src="https://git-scm.com/images/logos/downloads/Git-Icon-1788C.png"
              alt="git repository icon"
            />
            {git_repo_owner}/{git_repo_name}
            <RepoLink
              onClick={e => {
                e.stopPropagation();
                window.open(`https://github.com/${git_repo_owner}/${git_repo_name}`, "_blank")
              }}
            >
              <i className="material-icons">open_in_new</i>
            </RepoLink>
          </RepoName>
          <Status>
            {deployment_count > 0 ? (
              <>
                <StatusDot status={last_deployment_status} />
                Last PR status was "{capitalize(last_deployment_status || "")}"
                <Dot>•</Dot>
              </>
            ) : null}
            {deployment_count > 0 ? (
              <span>
                {deployment_count || 0}{" "}
                pull {deployment_count > 1 ? "requests" : "request"} deployed
              </span>
            ) : (
              <span>
                There is no pull request deployed for this environment
              </span>
            )}
          </Status>
        </DataContainer>
        <OptionWrapper>
          <Options.Dropdown expandIcon="more_vert" shrinkIcon="more_vert">
            <Options.Option onClick={() => setShowDeleteModal(true)}>
              <i className="material-icons">delete</i> Delete
            </Options.Option>
          </Options.Dropdown>
        </OptionWrapper>
      </EnvironmentCardWrapper>
    </>
  );
};

export default EnvironmentCard;

const OptionWrapper = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
`;

const RepoLink = styled.div`
  width: 22px;
  height: 22px;
  border-radius: 50px;
  margin-left: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  :hover {
    > i {
      color: #ffffff;
    }
  }

  > i {
    color: #ffffff99;
    font-size: 16px;
  }
`;

const EnvironmentCardWrapper = styled.div`
  display: flex;
  background: #2b2e3699;
  justify-content: space-between;
  border-radius: 5px;
  cursor: pointer;
  height: 75px;
  padding: 12px;
  padding-left: 14px;
  border: 1px solid #ffffff0f;

  :hover {
    border: 1px solid #ffffff3c;
  }
  animation: fadeIn 0.5s;
  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }
`;

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
`;

const RepoName = styled.div`
  display: flex;
  font-size: 14px;
  font-weight: 500;
  align-items: center;
`;

const Status = styled.span`
  font-size: 13px;
  display: flex;
  align-items: center;
  min-height: 17px;
  color: #a7a6bb;
  margin-top: 10px;
`;

const StatusDot = styled.div`
  width: 8px;
  height: 8px;
  margin-right: 15px;
  background: ${(props: { status: string }) =>
    props.status === "created"
      ? "#4797ff"
      : props.status === "failed"
      ? "#ed5f85"
      : props.status === "completed"
      ? "#00d12a"
      : "#f5cb42"};
  border-radius: 20px;
  margin-left: 4px;
`;

const Icon = styled.img`
  width: 18px;
  height: 18px;
  margin-right: 12px;
`;

const Button = styled.button`
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  margin-top: 13px;
  cursor: pointer;
  font-family: "Work Sans", sans-serif;
  border-radius: 5px;
  color: white;
  height: 35px;
  padding: 10px 16px;
  font-weight: 500;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  box-shadow: 0 5px 8px 0px #00000010;
  cursor: pointer;
  border: none;
  :not(:last-child) {
    margin-right: 10px;
  }
`;

const DeleteButton = styled(Button)`
  ${({ disabled }: { disabled: boolean }) => {
    if (disabled) {
      return `
      background: #aaaabbee;
      :hover {
        background: #aaaabbee;
      }    
      `;
    }

    return `
      background: #dd4b4b;
      :hover {
        background: #b13d3d;
      }`;
  }}
`;

const CancelButton = styled(Button)`
  background: #616feecc;
  :hover {
    background: #505edddd;
  }
`;

const ActionWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const Warning = styled.div`
  font-size: 13px;
  display: flex;
  border-radius: 3px;
  width: calc(100%);
  margin-top: 18px;
  margin-left: 2px;
  line-height: 1.4em;
  align-items: center;
  color: white;
  > i {
    margin-right: 10px;
    font-size: 18px;
  }
  color: ${(props: { highlight: boolean; makeFlush?: boolean }) =>
    props.highlight ? "#f5cb42" : ""};
`;

const Dot = styled.div`
  margin-right: 9px;
  margin-left: 9px;
`;