import React from "react";
import { ScrollView, StyleSheet } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { savePreferences } from "../api/dating";
import { PreferencesForm } from "../components/forms/PreferencesForm";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { spacing } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Preferences">;

export default function PreferencesScreen({ navigation }: Props) {
  const { self, refreshSelf } = useAuth();
  const save = useMutation({
    mutationFn: savePreferences,
    onSuccess: async () => {
      await refreshSelf();
      showToast("Préférences enregistrées", "success");
      navigation.goBack();
    }
  });
  return (
    <ScrollView contentContainerStyle={styles.content} testID="preferences-screen">
      <PreferencesForm
        busy={save.isPending}
        error={save.isError ? describeError(save.error) : null}
        initial={self?.preferences}
        onSubmit={(values) => save.mutate(values)}
        submitLabel="Enregistrer"
      />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.xl }
});
