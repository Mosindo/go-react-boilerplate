import React, { useEffect, useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { savePreferences } from "../api/profile";
import type { Preferences } from "../api/types";
import { DEFAULT_PREFERENCES, PreferencesForm } from "../components/PreferencesForm";
import { ScreenBar } from "../components/ScreenBar";
import { usePreferencesQuery } from "../hooks/useProfileData";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Notice, spacing } from "../shared/ui";

export default function PreferencesScreen() {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const query = usePreferencesQuery();
  const [value, setValue] = useState<Preferences>(DEFAULT_PREFERENCES);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (query.data) {
      setValue(query.data);
    }
  }, [query.data]);

  const save = async () => {
    if (value.interestedIn.length === 0) {
      setError("Choisissez au moins une option.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await savePreferences(value);
      await queryClient.invalidateQueries({ queryKey: ["preferences"] });
      // The deck depends on these rules: rebuild it.
      await queryClient.invalidateQueries({ queryKey: ["discover"] });
      navigation.goBack();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ScreenContainer contentMaxWidth={520} testID="preferences-screen">
      <ScreenBar title="Préférences de rencontre" />
      {query.isLoading ? (
        <LoadingView label="Chargement…" />
      ) : query.isError ? (
        <ErrorView actionLabel="Réessayer" message={errorMessage(query.error)} onAction={() => void query.refetch()} />
      ) : (
        <ScrollView contentContainerStyle={styles.content}>
          <PreferencesForm onChange={setValue} value={value} />
          {error ? <Notice description={error} title="Enregistrement impossible" tone="danger" /> : null}
          <View>
            <Button label="Enregistrer" loading={saving} onPress={save} size="lg" testID="preferences-save" />
          </View>
        </ScrollView>
      )}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({ content: { gap: spacing.xl, paddingBottom: spacing.xl } });
