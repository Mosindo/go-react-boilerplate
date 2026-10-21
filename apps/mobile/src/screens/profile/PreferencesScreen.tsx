import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useState } from "react";
import { fetchPreferences, savePreferences } from "../../api/profile";
import type { Preferences } from "../../api/types";
import { PreferencesFields } from "../../components/PreferencesFields";
import { keys } from "../../hooks/queries";
import type { RootStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView, showToast } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Button, Notice } from "../../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Preferences">;

export function PreferencesScreen({ navigation }: Props) {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.preferences, queryFn: fetchPreferences });
  const [value, setValue] = useState<Preferences | null>(null);

  useEffect(() => {
    if (query.data && !value) setValue(query.data);
  }, [query.data, value]);

  const save = useMutation({
    mutationFn: (prefs: Preferences) => savePreferences(prefs),
    onSuccess: async (prefs) => {
      queryClient.setQueryData(keys.preferences, prefs);
      await queryClient.invalidateQueries({ queryKey: keys.discover });
      showToast("Préférences enregistrées.", "success");
      navigation.goBack();
    }
  });

  if (query.isPending || !value)
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  if (query.isError)
    return (
      <ScreenContainer>
        <ErrorView message={query.error.message} onRetry={() => void query.refetch()} />
      </ScreenContainer>
    );

  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="preferences-screen">
      {save.isError ? <Notice message={save.error instanceof Error ? save.error.message : "Enregistrement impossible."} /> : null}
      <PreferencesFields value={value} onChange={setValue} />
      <Button testID="preferences-save" label="Enregistrer" loading={save.isPending} onPress={() => save.mutate(value)} />
    </ScreenContainer>
  );
}
