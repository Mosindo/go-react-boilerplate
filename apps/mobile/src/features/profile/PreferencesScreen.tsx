import React, { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { profileApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { keys } from "../../providers/queryClient";
import { Button, Header, LoadingState, Screen, showToast, Text } from "../../ui";
import { swipeStore } from "../discovery/swipeStore";
import { PreferencesFields, type PreferencesValue } from "./ProfileFields";
import { useMyProfile } from "./queries";

type Props = NativeStackScreenProps<AppStackParamList, "Preferences">;

export function PreferencesScreen({ navigation }: Props) {
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const [value, setValue] = useState<PreferencesValue | null>(null);

  useEffect(() => {
    if (profile.data?.preferences && !value) {
      setValue(profile.data.preferences);
    }
  }, [profile.data, value]);

  const save = useMutation({
    mutationFn: (prefs: PreferencesValue) => profileApi.savePreferences(prefs),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      queryClient.removeQueries({ queryKey: keys.discovery });
      swipeStore.reset();
      showToast("Préférences enregistrées.", "success");
      navigation.goBack();
    }
  });

  return (
    <Screen scroll header={<Header back title="Préférences de rencontre" />}>
      {!value ? (
        <LoadingState />
      ) : (
        <>
          <PreferencesFields error={value.interestedIn.length ? null : "Choisis au moins une option."} onChange={setValue} value={value} />
          <Text tone="subtle" variant="caption">
            Les préférences sont mutuelles : tu vois les personnes qui correspondent à tes critères et dont tu corresponds aux leurs.
          </Text>
          {save.isError ? <Text tone="danger">{errorMessage(save.error)}</Text> : null}
          <Button disabled={!value.interestedIn.length} label="Enregistrer" loading={save.isPending} onPress={() => save.mutate(value)} />
        </>
      )}
    </Screen>
  );
}
