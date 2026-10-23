import React from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { savePreferences } from "../../api";
import type { Preferences } from "../../api/types";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { usePreferences } from "../../queries";
import { queryKeys } from "../../session/queryKeys";
import { ErrorState, LinkRow, LoadingState, Screen, SwitchRow, useToast } from "../../ui";
import type { MainStackParams } from "../navTypes";

type Props = NativeStackScreenProps<MainStackParams, "Privacy">;

export default function PrivacyScreen({ navigation }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = usePreferences();

  const save = useMutation({
    mutationFn: (value: Preferences) => savePreferences(value),
    onMutate: async (value) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.preferences });
      const previous = queryClient.getQueryData<Preferences>(queryKeys.preferences);
      queryClient.setQueryData(queryKeys.preferences, value);
      return { previous };
    },
    onError: (error, _value, context) => {
      if (context?.previous) queryClient.setQueryData(queryKeys.preferences, context.previous);
      toast.error(errorMessage(error));
    },
    onSuccess: () => {
      toast.success(t.common.saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
    }
  });

  if (query.isPending) return <LoadingState />;
  if (query.isError) return <ErrorState onRetry={() => query.refetch()} title={t.errors.generic} />;
  const prefs = query.data;

  return (
    <Screen scroll testID="privacy-screen">
      <SwitchRow
        hint={t.privacy.discoverableHint}
        label={t.privacy.discoverable}
        onChange={(discoverable) => save.mutate({ ...prefs, discoverable })}
        testID="privacy-discoverable"
        value={prefs.discoverable}
      />
      <SwitchRow
        hint={t.privacy.showDistanceHint}
        label={t.privacy.showDistance}
        onChange={(showDistance) => save.mutate({ ...prefs, showDistance })}
        testID="privacy-show-distance"
        value={prefs.showDistance}
      />
      <LinkRow label={t.privacy.blocked} onPress={() => navigation.navigate("Blocked")} />
    </Screen>
  );
}
