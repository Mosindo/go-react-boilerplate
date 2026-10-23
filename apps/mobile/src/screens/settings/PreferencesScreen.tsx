import React, { useEffect, useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { savePreferences } from "../../api";
import type { Preferences } from "../../api/types";
import { GenderMultiPicker } from "../../features/ProfileFields";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { usePreferences } from "../../queries";
import { queryKeys } from "../../session/queryKeys";
import { Button, ErrorState, LoadingState, Screen, Stepper, Text, useToast } from "../../ui";
import type { MainStackParams } from "../navTypes";

type Props = NativeStackScreenProps<MainStackParams, "Preferences">;

export default function PreferencesScreen({ navigation }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = usePreferences();
  const [draft, setDraft] = useState<Preferences | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (query.data && !draft) setDraft(query.data);
  }, [query.data, draft]);

  const save = useMutation({
    mutationFn: (value: Preferences) => savePreferences(value),
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.preferences, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      toast.success(t.common.saved);
      navigation.goBack();
    },
    onError: (err) => toast.error(errorMessage(err))
  });

  if (query.isPending || (!draft && !query.isError)) return <LoadingState />;
  if (query.isError || !draft)
    return <ErrorState onRetry={() => query.refetch()} title={t.errors.generic} />;

  const patch = (next: Partial<Preferences>) => setDraft({ ...draft, ...next });
  const submit = () => {
    if (draft.interestedIn.length === 0) {
      setError(t.preferences.interestedRequired);
      return;
    }
    setError(null);
    save.mutate(draft);
  };

  return (
    <Screen scroll testID="preferences-screen">
      <GenderMultiPicker
        error={error}
        label={t.preferences.interestedIn}
        onChange={(interestedIn) => patch({ interestedIn })}
        testPrefix="pref-interested"
        value={draft.interestedIn}
      />
      <Stepper
        label={t.preferences.minAge}
        max={draft.maxAge}
        min={18}
        onChange={(minAge) => patch({ minAge })}
        testID="pref-min-age"
        unit={t.common.yearsUnit}
        value={draft.minAge}
      />
      <Stepper
        label={t.preferences.maxAge}
        max={99}
        min={draft.minAge}
        onChange={(maxAge) => patch({ maxAge })}
        testID="pref-max-age"
        unit={t.common.yearsUnit}
        value={draft.maxAge}
      />
      <Stepper
        label={t.preferences.distance}
        max={500}
        min={1}
        onChange={(maxDistanceKm) => patch({ maxDistanceKm })}
        step={5}
        testID="pref-distance"
        unit="km"
        value={draft.maxDistanceKm}
      />
      <Text tone="muted" variant="caption">
        {t.preferences.distanceHint}
      </Text>
      <Button
        label={t.common.save}
        loading={save.isPending}
        onPress={submit}
        testID="pref-save-button"
      />
    </Screen>
  );
}
