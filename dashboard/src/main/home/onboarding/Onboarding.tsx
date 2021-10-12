import React, { useContext, useEffect } from "react";
import { useLocation } from "react-router";
import { Context } from "shared/Context";
import { useRouting } from "shared/routing";
import styled from "styled-components";
import { useSnapshot } from "valtio";
import { devtools } from "valtio/utils";
import Routes from "./Routes";
import { OFState } from "./state";
import { useSteps } from "./state/StepHandler";

const Onboarding = () => {
  useSteps();

  useEffect(() => {
    let sub = devtools(OFState, "Onboarding flow state");
    return () => {
      sub();
    };
  }, []);

  // useEffect(() => {
  //   OFState.actions.initializeState(context.currentProject?.id);
  //   return () => {
  //     OFState.actions.clearState();
  //   };
  // }, [context.currentProject?.id]);

  // useEffect(() => {
  //   if (snap.StepHandler.finishedOnboarding) {
  //     OFState.actions.clearState();
  //     pushFiltered("/dashboard", []);
  //   } else if (snap.StepHandler?.currentStep?.url !== location.pathname) {
  //     pushFiltered(snap.StepHandler.currentStep.url, []);
  //   }
  // }, [
  //   location.pathname,
  //   snap.StepHandler?.currentStep?.url,
  //   snap.StepHandler?.finishedOnboarding,
  // ]);

  return (
    <StyledOnboarding>
      <Routes />
    </StyledOnboarding>
  );
};

export default Onboarding;

const StyledOnboarding = styled.div`
  width: calc(90% - 130px);
  min-width: 300px;
  position: relative;
  margin-top: calc(50vh - 340px);
`;
