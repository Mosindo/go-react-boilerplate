import React from "react";
import { Loader } from "../ui";

type LoadingViewProps = {
  fullScreen?: boolean;
  label?: string;
};

export function LoadingView({ fullScreen = false, label }: LoadingViewProps) {
  return <Loader fullScreen={fullScreen} label={label} />;
}
