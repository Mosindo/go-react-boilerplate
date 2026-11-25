import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/profile";
import type { Preferences } from "../../api/types";
import { PreferencesFields } from "../../components/PreferencesFields";
import { validatePreferences } from "../../components/profileForm";
import { useMyProfile, useSetMyProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { Button, Header, LoadingState, Screen, Text, showToast } from "../../ui";

export function PreferencesScreen({ navigation }: AppScreenProps<"Preferences">) {
  const me = useMyProfile();
  const setMyProfile = useSetMyProfile();
  const [draft, setDraft] = useState<Preferences | null>(null);
  const [problem, setProblem] = useState<string | null>(null);
  const value = draft ?? me.data?.preferences;

  const save = useMutation({
    mutationFn: async () => {
      if (!value) return null;
      const issue = validatePreferences(value);
      setProblem(issue);
      return issue ? null : profileApi.updatePreferences(value);
    },
    onSuccess: (profile) => {
      if (profile) {
        setMyProfile(profile);
        showToast("Préférences enregistrées.", "success");
        navigation.goBack();
      }
    }
  });

  return (
    <Screen
      scroll
      testID="preferences-screen"
      footer={
        <>
          {save.error ? (
            <Text variant="small" color="danger" align="center">
              {errorMessage(save.error)}
            </Text>
          ) : null}
          <Button label="Enregistrer" onPress={() => save.mutate()} loading={save.isPending} disabled={!value} testID="preferences-save" />
        </>
      }
    >
      <Header title="Préférences de rencontre" onBack={navigation.goBack} />
      {value ? <PreferencesFields value={value} onChange={setDraft} error={problem} /> : <LoadingState />}
    </Screen>
  );
}
