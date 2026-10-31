import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { savePreferences } from "../api/dating";
import type { OwnProfile } from "../api/types";
import { PreferencesForm } from "../components/PreferencesForm";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Button, Notice } from "../shared/ui";

type Props = NativeStackScreenProps<MainStackParamList, "Preferences">;

function Editor({ profile, onDone }: { profile: OwnProfile; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState(profile.preferences);
  const [error, setError] = useState<string | null>(null);
  const save = useMutation({
    mutationFn: () => savePreferences(draft),
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.feed });
      onDone();
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Enregistrement impossible.")
  });
  const submit = () => {
    if (draft.interestedIn.length === 0) {
      setError("Choisissez au moins un genre à rencontrer.");
      return;
    }
    setError(null);
    save.mutate();
  };
  return (
    <Screen scroll>
      <PreferencesForm onChange={setDraft} value={draft} />
      {error ? <Notice title={error} tone="danger" /> : null}
      <Button label="Enregistrer" loading={save.isPending} onPress={submit} />
    </Screen>
  );
}

export default function PreferencesScreen({ navigation }: Props) {
  const { data, isLoading, isError, refetch } = useOwnProfile();
  if (isLoading)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (isError || !data)
    return (
      <Screen>
        <ErrorView actionLabel="Réessayer" message="Préférences indisponibles." onAction={() => void refetch()} />
      </Screen>
    );
  return <Editor onDone={() => navigation.goBack()} profile={data} />;
}
