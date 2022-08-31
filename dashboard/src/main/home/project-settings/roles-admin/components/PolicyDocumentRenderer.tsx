import _, { capitalize, get, set } from "lodash";
import React, { useContext, useEffect, useRef } from "react";
import api from "shared/api";
import {
  POLICY_HIERARCHY_TREE,
  populatePolicy,
} from "shared/auth/authorization-helpers";
import { PolicyDocType, ScopeType, Verbs } from "shared/auth/types";
import { Context } from "shared/Context";
import styled from "styled-components";

type VerbStore = {
  data: PolicyDocType;
  handleChangeVerbs: (path: string, values: string[]) => void;
};

// Store that will save the current state of the policy document,
// only changes applied for the policy document are verb changes.
const Store = React.createContext<VerbStore>({
  data: null,
  handleChangeVerbs: () => {},
});

const PolicyDocumentRenderer = ({
  value,
  onChange,
  readOnly,
}: {
  value: PolicyDocType;
  onChange: (data: PolicyDocType) => void;
  readOnly?: boolean;
}) => {
  const { currentProject, setCurrentOverlay } = useContext(Context);
  const [scopeHierarchy, setScopeHierarchy] = React.useState<any>(null);
  const emptyPolicyDoc = useRef(
    populatePolicy({
      scope: "project",
      verbs: [],
    })
  );

  useEffect(() => {
    if (!scopeHierarchy) {
      return;
    }

    emptyPolicyDoc.current = populatePolicy(
      {
        scope: "project",
        verbs: [],
      },
      scopeHierarchy
    );
  }, [scopeHierarchy]);

  useEffect(() => {
    api
      .getScopeHierarchy("<token>", {}, { project_id: currentProject.id })
      .then((res) => {
        setScopeHierarchy(res.data);
        const newPolicyDoc = structuredClone(value);

        onChange(populatePolicy(newPolicyDoc, res.data));
      });
  }, [currentProject?.id]);

  const handleChangeVerbs = (verbsPath: string, verbs: Verbs[]) => {
    const newPolicyDoc = structuredClone(value) as PolicyDocType;
    const pathToChildren = verbsPath
      .split(".")
      .slice(0, -1)
      .concat(["children"])
      .join(".");

    const isReadRemovedTransitive =
      !verbs.includes("get") &&
      Object.keys(_.get(newPolicyDoc, pathToChildren, {})).length;

    if (isReadRemovedTransitive) {
      const emptyPolicyDocumentForChildren = get(
        emptyPolicyDoc.current,
        pathToChildren
      );

      set(newPolicyDoc, pathToChildren, emptyPolicyDocumentForChildren);
    }

    set(newPolicyDoc, verbsPath, verbs);

    if (isReadRemovedTransitive) {
      setCurrentOverlay({
        message: `Dummy text?`,
        onYes: () => {
          onChange(newPolicyDoc);
          setCurrentOverlay(null);
        },
        onNo: () => setCurrentOverlay(null),
      });
    } else {
      onChange(newPolicyDoc);
    }
  };

  if (!scopeHierarchy) {
    return (
      <>
        <h1>Loading...</h1>
      </>
    );
  }

  return (
    <Store.Provider
      value={{ data: populatePolicy(value, scopeHierarchy), handleChangeVerbs }}
    >
      {RenderComponents(readOnly, value, scopeHierarchy)}
    </Store.Provider>
  );
};

export default PolicyDocumentRenderer;

const RenderComponents = (
  readOnly: boolean,
  policyDocument: PolicyDocType,
  tree = POLICY_HIERARCHY_TREE,
  dataPath = "",
  anidationLevel = 0
) => {
  const scope = policyDocument.scope;

  const currTree = tree[scope];
  const treeKeys = Object.keys(currTree) as Array<ScopeType>;

  let components: React.ReactElement[] = [];

  const newDataPath = anidationLevel === 0 ? "" : dataPath + "." + scope;

  const verbsPath = newDataPath === "" ? "verbs" : newDataPath + ".verbs";

  const childrenPath =
    newDataPath === "" ? "children" : newDataPath + ".children";

  for (const child of treeKeys) {
    let childPolicy = policyDocument.children[child];

    if (!childPolicy) {
      continue;
    }

    const children = RenderComponents(
      readOnly,
      childPolicy,
      currTree,
      childrenPath,
      anidationLevel + 1
    );
    components = [...components, children];
  }

  const Component = (
    <>
      <ScopePermissionsHandler
        name={scope}
        parent={dataPath.split(".").slice(0, -1).join(".")}
        dataPath={verbsPath}
        readOnly={readOnly}
        anidationLevel={anidationLevel}
      />
      {components.map((c) => c)}
    </>
  );
  return Component;
};

const Card = styled.div<{ anidationLevel: number }>`
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-around;
  background-color: #2b2e3699;
  margin-left: ${({ anidationLevel }) => `${anidationLevel * 20}px`};
  margin-bottom: 15px;
`;

const ScopePermissionsHandler = ({
  anidationLevel,
  parent,
  name,
  dataPath,
  readOnly,
}: {
  anidationLevel: number;
  parent: string;
  name: string;
  dataPath: string;
  readOnly: boolean;
}) => {
  const { handleChangeVerbs, data } = React.useContext(Store);
  const { setCurrentError } = React.useContext(Context);

  const verbs = get(data, dataPath);

  const onChange = (newVerbs: Verbs[]) => {
    const pathToParentsVerbs = parent.split(".").concat("verbs").join(".");

    const isActionAllowed =
      anidationLevel === 0
        ? true
        : get(data, pathToParentsVerbs, []).includes("get");

    if (isActionAllowed) {
      handleChangeVerbs(dataPath, newVerbs);
      return;
    }

    setCurrentError(
      "This action is not allowed since the parent does not have read permissions enabled."
    );

    // show something to tell user
  };

  return (
    <Card anidationLevel={anidationLevel}>
      {name}
      {readOnly ? null : <Select values={verbs} onChange={onChange} />}
    </Card>
  );
};

type SelectProps = {
  values: Verbs[];
  onChange: (newVerbs: Verbs[]) => void;
  disabled?: boolean;
};

const Select = ({ values, onChange }: SelectProps) => {
  const options = ["create", "read", "update", "delete"] as const;
  const [open, setOpen] = React.useState(false);

  const handleChange = (opt: typeof options[number], add: boolean) => {
    const verbs: Verbs[] = opt === "read" ? ["get", "list"] : [opt];

    if (add) {
      handleAdd(verbs);
    } else {
      handleDelete(verbs);
    }
  };

  const handleAdd = (verbs: Verbs[]) => {
    const newValues = [...verbs, ...values];
    onChange(newValues);
  };

  const handleDelete = (verbs: Verbs[]) => {
    const newValues = values.filter((val) => !verbs.includes(val));

    onChange(newValues);
  };

  return (
    <div>
      <i onClick={() => setOpen(!open)}>{capitalize(values.join(", "))}</i>
      <div>
        {options.map((opt) => {
          const isChecked =
            opt === "read"
              ? values.find((val) => val === "get")
              : values.find((val) => val === opt);
          return (
            <div>
              <input
                type="checkbox"
                name={opt}
                checked={!!isChecked}
                onChange={(e) => handleChange(opt, e.target.checked)}
              />
              <label htmlFor={opt}>{capitalize(opt)}</label>
            </div>
          );
        })}
      </div>
    </div>
  );
};
