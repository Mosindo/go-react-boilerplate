import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi, queryKeys, type Preferences, type ProfileState } from "../../api";
import { showToast } from "../../shared/feedback";
import { Button, ErrorState, Screen, Skeleton, Text } from "../../shared/ui";
import type { MainStackParamList } from "../../navigation/types";
import { bumpDeckEpoch } from "../discovery/deckEpoch";
import { DEFAULT_PREFERENCES, PreferencesForm } from "./components/PreferencesForm";
import { useProfileState } from "./hooks";

type Props = NativeStackScreenProps<MainStackParamList, "EditPreferences">;

export function EditPreferencesScreen(props: Props) {
  const query = useProfileState();
  if (query.isError) {
    return (
      <Screen>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  if (!query.data) {
    return (
      <Screen>
        <Skeleton height={160} />
      </Screen>
    );
  }
  return <PreferencesEditor {...props} initial={query.data.preferences ?? DEFAULT_PREFERENCES} />;
}

function PreferencesEditor({ navigation, initial }: Props & { initial: Preferences }) {
  const queryClient = useQueryClient();
  const [value, setValue] = useState(initial);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (preferences) => {
      queryClient.setQueryData<ProfileState>(queryKeys.profile, (state) => (state ? { ...state, preferences } : state));
      bumpDeckEpoch();
      showToast("Préférences enregistrées.");
      navigation.goBack();
    }
  });

  const submit = () => {
    if (value.interestedIn.length === 0) {
      setError("Choisissez au moins une option.");
      return;
    }
    setError(null);
    save.mutate(value);
  };

  return (
    <Screen
      footer={<Button label="Enregistrer" loading={save.isPending} onPress={submit} testID="preferences-save" />}
      scroll
      testID="preferences-screen"
    >
      <PreferencesForm onChange={setValue} value={value} />
      {error ? (
        <Text color="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </Screen>
  );
}
