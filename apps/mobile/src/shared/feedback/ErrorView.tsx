import { errorMessage } from "../../lib/errors";
import { EmptyView } from "./EmptyView";

export function ErrorView({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return <EmptyView icon="cloud-offline-outline" title="Oups…" message={errorMessage(error)} actionLabel={onRetry ? "Réessayer" : undefined} onAction={onRetry} />;
}
