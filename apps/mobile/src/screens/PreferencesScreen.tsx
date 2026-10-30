import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { profileApi } from "../api/endpoints";
import type { Preferences } from "../api/types";
import { PreferencesForm, validatePreferences } from "../components/PreferencesForm";
import { errorMessage, keys, useProfile } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { discoverStore } from "../state/discoverStore";
import { Button, ErrorView, LoadingView, Screen, Text, spacing, useToast } from "../ui";

type Props = NativeStackScreenProps<MainStackParamList, "Preferences">;

function Form({ initial, onSaved }: { initial: Preferences; onSaved: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [prefs, setPrefs] = useState(initial);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    const invalid = validatePreferences(prefs);
    if (invalid) return setError(invalid);
    setError(null);
    setBusy(true);
    try {
      await profileApi.savePreferences(prefs);
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      discoverStore.refresh();
      toast.show("Préférences enregistrées.");
      onSaved();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <View style={styles.form}>
      <PreferencesForm onChange={setPrefs} value={prefs} />
      {error ? (
        <Text accessibilityRole="alert" tone="danger">
          {error}
        </Text>
      ) : null}
      <Button label="Enregistrer" loading={busy} onPress={save} testID="prefs-save" />
    </View>
  );
}

export default function PreferencesScreen({ navigation }: Props) {
  const query = useProfile();
  if (query.isLoading) return <LoadingView />;
  if (!query.data) return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  return (
    <Screen scroll testID="preferences-screen">
      <Form initial={query.data.preferences} onSaved={() => navigation.goBack()} />
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.xl, paddingTop: spacing.lg } });
