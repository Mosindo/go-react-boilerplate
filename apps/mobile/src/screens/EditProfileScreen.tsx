import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { saveProfile } from "../api/dating";
import type { OwnProfile } from "../api/types";
import { ProfileForm, toInput } from "../components/ProfileForm";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Button, Notice } from "../shared/ui";

type Props = NativeStackScreenProps<MainStackParamList, "EditProfile">;

function Editor({ profile, onDone }: { profile: OwnProfile; onDone: () => void }) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState(() => toInput(profile));
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () => saveProfile(draft),
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.feed });
      onDone();
    },
    onError: (err) => setError(err instanceof ApiError ? err.message : "Enregistrement impossible.")
  });

  const submit = () => {
    if (!draft.firstName.trim()) {
      setError("Indiquez votre prénom.");
      return;
    }
    setError(null);
    save.mutate();
  };

  return (
    <Screen scroll>
      <ProfileForm onChange={setDraft} value={draft} />
      {error ? <Notice title={error} tone="danger" /> : null}
      <Button label="Enregistrer" loading={save.isPending} onPress={submit} testID="save-profile" />
    </Screen>
  );
}

export default function EditProfileScreen({ navigation }: Props) {
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
        <ErrorView actionLabel="Réessayer" message="Profil indisponible." onAction={() => void refetch()} />
      </Screen>
    );
  return <Editor onDone={() => navigation.goBack()} profile={data} />;
}
