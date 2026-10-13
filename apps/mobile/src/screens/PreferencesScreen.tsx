import React, { useState } from "react";
import { ScrollView, StyleSheet } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { savePreferences } from "../api/platform";
import { errorMessage } from "../api/client";
import type { Preferences } from "../api/types";
import { DEFAULT_PREFERENCES, PreferencesFields } from "../features/profile/PreferencesFields";
import { useOwnProfile } from "../hooks/queries";
import { queryKeys } from "../lib/queryClient";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Notice, spacing } from "../shared/ui";

export default function PreferencesScreen() {
  const query = useOwnProfile();
  if (query.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (!query.data) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger vos préférences." onAction={() => void query.refetch()} />
      </ScreenContainer>
    );
  }
  return <PreferencesForm initial={query.data.preferences ?? DEFAULT_PREFERENCES} />;
}

function PreferencesForm({ initial }: { initial: Preferences }) {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const [value, setValue] = useState(initial);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () => savePreferences(value),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.ownProfile });
      showToast("Préférences enregistrées.");
      navigation.goBack();
    },
    onError: (e) => setError(errorMessage(e))
  });

  return (
    <ScreenContainer>
      <ScrollView contentContainerStyle={styles.scroll}>
        {error ? <Notice message={error} tone="danger" /> : null}
        <PreferencesFields onChange={setValue} value={value} />
        <Button
          disabled={value.interestedIn.length === 0}
          fullWidth
          label="Enregistrer"
          loading={save.isPending}
          onPress={() => save.mutate()}
          size="lg"
          testID="prefs-save"
        />
      </ScrollView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({ scroll: { gap: spacing.xl, paddingBottom: spacing.xxxl } });
