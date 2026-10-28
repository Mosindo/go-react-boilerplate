import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { savePreferences } from "../../api/profile";
import type { Preferences } from "../../api/types";
import { Button, ErrorState, LoadingState, PreferencesForm, Screen, Text, useToast } from "../../components";
import { usePreferences } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";
import { validateAgeRange } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

function Form({ initial, onDone }: { initial: Preferences; onDone: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [prefs, setPrefs] = useState(initial);
  const problem = prefs.interestedIn.length === 0 ? "Pick at least one option." : validateAgeRange(prefs.minAge, prefs.maxAge);
  const mutation = useMutation({
    mutationFn: () => savePreferences(prefs),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.preferences });
      await queryClient.invalidateQueries({ queryKey: keys.discover });
      toast.show("Preferences saved", "success");
      onDone();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });
  return (
    <Screen scroll testID="preferences-screen">
      <View style={{ gap: spacing.xl }}>
        <PreferencesForm value={prefs} onChange={setPrefs} />
        {problem ? <Text tone="danger">{problem}</Text> : null}
        <Button title="Save" onPress={() => mutation.mutate()} disabled={!!problem} loading={mutation.isPending} testID="preferences-save" />
      </View>
    </Screen>
  );
}

export function PreferencesScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "Preferences">) {
  const query = usePreferences();
  if (query.isPending) return <Screen><LoadingState /></Screen>;
  if (query.isError) return <Screen><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></Screen>;
  return <Form initial={query.data} onDone={() => navigation.goBack()} />;
}
