import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { savePreferences, saveProfile } from "../../api/profile";
import type { Preferences } from "../../api/types";
import { PhotoGrid } from "../../components/PhotoGrid";
import { PreferencesFields } from "../../components/PreferencesFields";
import { ProfileFields } from "../../components/ProfileFields";
import { keys } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { usePhotoManager } from "../../hooks/usePhotoManager";
import { getApproxLocation } from "../../lib/device";
import { defaultPreferences } from "../../lib/preferences";
import { draftToInput, emptyDraft, validateDraft, type DraftErrors, type ProfileDraft } from "../../lib/profileDraft";
import { showError } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Button, Notice, Text } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";

const STEPS = ["À propos de vous", "Vos goûts", "Vos préférences", "Vos photos"] as const;

/** Four short steps: about → tastes → preferences → photos. Profile is saved on step 2, photos on step 4. */
export function OnboardingScreen({ onDone }: { onDone: () => void }) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const { user, refreshUser, signOut } = useAuth();
  const [step, setStep] = useState(user?.hasProfile ? (user.photoCount < 1 ? 3 : 2) : 0);
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [errors, setErrors] = useState<DraftErrors>({});
  const [prefs, setPrefs] = useState<Preferences>(defaultPreferences);
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const manager = usePhotoManager(step === 3);
  const photos = manager.photos;

  const patch = (p: Partial<ProfileDraft>) =>
    setDraft((d) => ({ ...d, ...p, clearLocation: p.latitude !== undefined ? false : (p.clearLocation ?? d.clearLocation) }));

  const useLocation = async () => {
    setLocating(true);
    const loc = await getApproxLocation();
    setLocating(false);
    if (!loc) return showError("Position indisponible. Vous pouvez saisir votre ville à la main.");
    patch({ latitude: loc.latitude, longitude: loc.longitude, city: draft.city || loc.city || "" });
  };

  const saveProfileMutation = useMutation({
    mutationFn: () => saveProfile(draftToInput(draft)),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      setStep(2);
    },
    onError: (err) => setError(err instanceof Error ? err.message : "Enregistrement impossible.")
  });

  const savePrefsMutation = useMutation({
    mutationFn: () => savePreferences(prefs),
    onSuccess: () => setStep(3),
    onError: (err) => setError(err instanceof Error ? err.message : "Enregistrement impossible.")
  });

  const next = () => {
    setError(null);
    if (step === 0) {
      const found = validateDraft(draft);
      setErrors(found);
      if (found.firstName || found.gender || found.bio || found.city) return;
      setStep(1);
    } else if (step === 1) {
      const found = validateDraft(draft);
      setErrors(found);
      if (Object.keys(found).length > 0) return;
      saveProfileMutation.mutate();
    } else if (step === 2) {
      savePrefsMutation.mutate();
    }
  };

  const finish = async () => {
    const me = await refreshUser();
    if (!me || me.photoCount < 1) return setError("Ajoutez au moins une photo pour continuer.");
    onDone();
  };

  return (
    <ScreenContainer scroll testID="onboarding-screen" contentStyle={{ gap: spacing.lg }}>
      <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((label, i) => (
          <View key={label} style={[styles.bar, { backgroundColor: i <= step ? theme.primary : theme.border }]} />
        ))}
      </View>
      <Text variant="heading" accessibilityRole="header" testID="onboarding-title">
        {STEPS[step]}
      </Text>
      {error ? <Notice message={error} /> : null}

      {step === 0 ? (
        <ProfileFields part="about" draft={draft} errors={errors} onChange={patch} onUseLocation={useLocation} locating={locating} />
      ) : null}
      {step === 1 ? <ProfileFields part="more" draft={draft} errors={errors} onChange={patch} onUseLocation={useLocation} /> : null}
      {step === 2 ? (
        <>
          <Text tone="muted">Vous pourrez les modifier à tout moment. Vos préférences ne sont jamais visibles par les autres.</Text>
          <PreferencesFields value={prefs} onChange={setPrefs} />
        </>
      ) : null}
      {step === 3 ? (
        <>
          <Text tone="muted">
            Ajoutez au moins une photo (6 maximum). Elles sont recadrées, compressées et nettoyées de leurs métadonnées.
          </Text>
          <PhotoGrid
            photos={photos}
            busy={manager.busy}
            onAdd={() => void manager.add()}
            onReplace={(p) => void manager.replace(p)}
            onDelete={(p) => void manager.remove(p)}
            onMakeMain={(p) => manager.move(p, 0)}
            onMove={(p, dir) => manager.move(p, p.position + dir)}
          />
        </>
      ) : null}

      <View style={styles.actions}>
        {step < 3 ? (
          <Button
            testID="onboarding-next"
            label="Continuer"
            loading={saveProfileMutation.isPending || savePrefsMutation.isPending}
            onPress={next}
          />
        ) : (
          <Button testID="onboarding-finish" label="Terminer" disabled={photos.length < 1} onPress={finish} />
        )}
        {step > 0 ? <Button label="Retour" variant="ghost" onPress={() => setStep((s) => Math.max(0, s - 1))} /> : null}
        <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </View>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: radius.pill },
  actions: { gap: spacing.sm, marginTop: spacing.md }
});
