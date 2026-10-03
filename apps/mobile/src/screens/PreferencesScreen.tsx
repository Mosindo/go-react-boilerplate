import React, { useEffect, useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage, getOwnProfile, queryKeys, savePreferences, type Preferences } from "../api/platform";
import { PreferencesFields, defaultPreferences } from "../components/PreferencesFields";
import { Screen } from "../components/Screen";
import type { RootStackParamList } from "../navigation/types";
import { LoadingView, showToast } from "../shared/feedback";
import { Button, Notice } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Preferences">;

export default function PreferencesScreen({ navigation }: Props) {
  const client = useQueryClient();
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
  const [value, setValue] = useState<Preferences>(defaultPreferences);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile.data && !ready) {
      setValue(profile.data.preferences);
      setReady(true);
    }
  }, [profile.data, ready]);

  const save = useMutation({
    mutationFn: () => savePreferences(value),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: queryKeys.profile });
      await client.invalidateQueries({ queryKey: queryKeys.discover });
      showToast("Preferences saved.", { tone: "success" });
      navigation.goBack();
    },
    onError: (e) => setError(errorMessage(e))
  });

  if (!ready) {
    return <LoadingView fullScreen label="Loading…" />;
  }
  return (
    <Screen testID="preferences-screen">
      {error ? <Notice description={error} title="Not saved" tone="danger" /> : null}
      <PreferencesFields onChange={setValue} value={value} />
      <Button label="Save preferences" loading={save.isPending} onPress={() => save.mutate()} size="lg" testID="preferences-save" />
    </Screen>
  );
}
