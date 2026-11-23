import React from "react";
import { EmptyView } from "./EmptyView";

type ErrorViewProps = {
  message?: string;
  onRetry?: () => void;
};

export function ErrorView({ message, onRetry }: ErrorViewProps) {
  return (
    <EmptyView
      actionLabel={onRetry ? "Réessayer" : undefined}
      glyph="!"
      message={message ?? "Vérifie ta connexion puis réessaie."}
      onAction={onRetry}
      testID="error-view"
      title="Impossible de charger"
    />
  );
}
