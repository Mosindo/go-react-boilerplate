import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { savePreferences } from "../api/platform";
import type { Preferences } from "../api/types";
import { DEFAULT_PREFERENCES, PreferencesForm } from "../components/PreferencesForm";
import { useMyProfile } from "../hooks/queries";
import { markDeckStale } from "../lib/deckSignal";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../realtime/cache";
import { ErrorView, LoadingView, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, Notice } from "../shared/ui";

export default function PreferencesScreen() {
  const { data, isLoading, isError, error, refetch } = useMyProfile();
  if (isLoading) return <LoadingView />;
  if (isError || !data) return <ErrorView error={error} onRetry={() => void refetch()} />;
  return <Editor initial={data.preferences ?? DEFAULT_PREFERENCES} />;
}

function Editor({ initial }: { initial: Preferences }) {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const [value, setValue] = useState(initial);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      await savePreferences(value);
      await queryClient.invalidateQueries({ queryKey: queryKeys.me });
      // The feed depends on preferences: the next visit to "Découvrir" refetches with the new criteria.
      markDeckStale();
      showSuccess("Préférences enregistrées.");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ScreenContainer scroll testID="preferences-screen">
      <Header title="Préférences" onBack={() => navigation.goBack()} />
      <PreferencesForm value={value} onChange={setValue} />
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="prefs-save" label="Enregistrer" onPress={save} loading={saving} />
    </ScreenContainer>
  );
}
