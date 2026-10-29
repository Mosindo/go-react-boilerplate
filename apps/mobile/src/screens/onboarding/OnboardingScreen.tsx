import React, { useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { PhotoGrid } from "../../components/PhotoGrid";
import { PreferencesForm } from "../../components/PreferencesForm";
import { ProfileForm } from "../../components/ProfileForm";
import { Button } from "../../components/Button";
import { ErrorView, LoadingView } from "../../components/StateViews";
import { usePreferences, useProfile } from "../../hooks/queries";
import { useAuth, useMe } from "../../state/AuthProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

const TITLES = [
  "Faisons connaissance",
  "Vos plus belles photos",
  "Qui souhaitez-vous rencontrer ?"
];
const SUBTITLES = [
  "Ces informations apparaissent sur votre profil.",
  "Ajoutez au moins une photo. Les profils avec plusieurs photos sont plus souvent aimés.",
  "Vous pourrez modifier ces préférences à tout moment."
];

export default function OnboardingScreen() {
  const theme = useTheme();
  const me = useMe();
  const { refreshMe, signOut } = useAuth();
  // Resume at the first step that is not done yet (e.g. after closing the app).
  const [step, setStep] = useState(() => (!me.hasProfile ? 0 : !me.hasPhotos ? 1 : 2));
  const profileQuery = useProfile();
  const preferencesQuery = usePreferences();

  const waitingForProfile = me.hasProfile && profileQuery.isLoading;
  const waitingForPreferences = step === 2 && me.hasPreferences && preferencesQuery.isLoading;
  const loadError =
    (me.hasProfile && profileQuery.isError) || (me.hasPreferences && preferencesQuery.isError);

  return (
    <SafeAreaView style={[styles.flex, { backgroundColor: theme.background }]}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur 3`}>
          {[0, 1, 2].map((i) => (
            <View
              key={i}
              style={[styles.dot, { backgroundColor: i <= step ? theme.primary : theme.border }]}
            />
          ))}
        </View>
        <Text accessibilityRole="header" style={[styles.title, { color: theme.text }]}>
          {TITLES[step]}
        </Text>
        <Text style={[styles.subtitle, { color: theme.textMuted }]}>{SUBTITLES[step]}</Text>

        {loadError ? (
          <ErrorView
            onRetry={() => {
              void profileQuery.refetch();
              void preferencesQuery.refetch();
            }}
          />
        ) : waitingForProfile || waitingForPreferences ? (
          <LoadingView />
        ) : step === 0 ? (
          <ProfileForm
            initial={profileQuery.data ?? null}
            onSaved={() => {
              void refreshMe();
              setStep(1);
            }}
            submitLabel="Continuer"
          />
        ) : step === 1 ? (
          <View>
            <PhotoGrid />
            <Button
              disabled={!me.hasPhotos}
              onPress={() => setStep(2)}
              style={styles.next}
              testID="photos-next"
              title="Continuer"
            />
            <Button onPress={() => setStep(0)} title="Retour" variant="ghost" />
          </View>
        ) : (
          <PreferencesForm
            initial={preferencesQuery.data ?? null}
            onSaved={() => void refreshMe()}
            submitLabel="Découvrir des profils"
          />
        )}

        <Button
          onPress={() => void signOut()}
          style={styles.signOut}
          title="Se déconnecter"
          variant="ghost"
        />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { padding: spacing.lg, paddingBottom: spacing.xxl },
  progress: { flexDirection: "row", gap: spacing.sm, marginBottom: spacing.xl },
  dot: { flex: 1, height: 4, borderRadius: 2 },
  title: { fontSize: 28, fontWeight: "700", letterSpacing: -0.5 },
  subtitle: {
    fontSize: fontSizes.md,
    lineHeight: 22,
    marginTop: spacing.sm,
    marginBottom: spacing.xl
  },
  next: { marginTop: spacing.xl },
  signOut: { marginTop: spacing.xl }
});
