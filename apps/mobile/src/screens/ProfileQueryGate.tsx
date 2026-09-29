import React, { type ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { messageFromError } from "../lib/errors";
import { ErrorView } from "../shared/feedback/ErrorView";
import { LoadingView } from "../shared/feedback/LoadingView";
import { ProfileSubScreen } from "./ProfileSubScreen";

type Props<T> = {
  query: UseQueryResult<T, Error>;
  title: string;
  onBack: () => void;
  children: (data: T) => ReactNode;
};

/** Renders loading / error states inside a sub-screen frame, then hands loaded data to `children`. */
export function ProfileQueryGate<T>({ children, onBack, query, title }: Props<T>) {
  if (query.isPending) {
    return (
      <ProfileSubScreen onBack={onBack} title={title}>
        <LoadingView label="Loading..." />
      </ProfileSubScreen>
    );
  }
  if (query.isError && query.data === undefined) {
    return (
      <ProfileSubScreen onBack={onBack} title={title}>
        <ErrorView message={messageFromError(query.error)} onAction={() => void query.refetch()} />
      </ProfileSubScreen>
    );
  }
  return <>{children(query.data as T)}</>;
}
