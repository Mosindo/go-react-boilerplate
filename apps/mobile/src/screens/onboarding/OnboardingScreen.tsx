import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { MyProfile } from "../../api/types";
import { InterestsPicker } from "../../features/InterestsPicker";
import { LocationControl } from "../../features/LocationControl";
import { PhotosManager } from "../../features/PhotosManager";
import { PreferencesForm } from "../../features/PreferencesForm";
import { ProfileForm } from "../../features/ProfileForm";
import { useAuth } from "../../hooks/useAuth";
import { useProfileMutation } from "../../hooks/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Screen, Text } from "../../ui";

const STEPS = [
  { title: "Faisons connaissance", subtitle: "Ces informations apparaîtront sur ton profil." },
  { title: "Qui aimerais-tu rencontrer ?", subtitle: "Tu pourras changer d'avis à tout moment." },
  { title: "Tes centres d'intérêt", subtitle: "Ils aident à trouver des points communs. Facultatif." },
  { title: "Près de chez toi", subtitle: "Pour te proposer des profils à une distance raisonnable." },
  { title: "Tes photos", subtitle: "Ajoute au moins une photo pour apparaître dans la découverte." }
];

function initialStep(p: MyProfile): number {
  if (!p.profile) {
    return 0;
  }
  if (!p.preferences) {
    return 1;
  }
  return p.photos.length > 0 ? 4 : 2;
}

/**
 * Shown while the profile is incomplete. The root navigator switches to the
 * main app automatically once the server reports `isComplete`.
 */
export function OnboardingScreen({ profile }: { profile: MyProfile }) {
  const { colors, radii } = useTheme();
  const { signOut } = useAuth();
  const [step, setStep] = useState(() => initialStep(profile));
  const next = () => setStep((s) => Math.min(s + 1, STEPS.length - 1));

  const saveProfile = useProfileMutation(profileApi.saveProfile);
  const savePrefs = useProfileMutation(profileApi.savePreferences);
  const saveInterests = useProfileMutation(profileApi.saveInterests);

  return (
    <Screen testID={`onboarding-step-${step}`}>
      <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View
            key={s.title}
            style={[
              styles.bar,
              { borderRadius: radii.pill, backgroundColor: i <= step ? colors.primary : colors.border }
            ]}
          />
        ))}
      </View>
      <View style={{ gap: 6 }}>
        <Text variant="overline" tone="primary">
          Étape {step + 1} sur {STEPS.length}
        </Text>
        <Text variant="title">{STEPS[step].title}</Text>
        <Text tone="muted">{STEPS[step].subtitle}</Text>
      </View>

      {step === 0 ? (
        <ProfileForm
          initial={profile.profile}
          submitLabel="Continuer"
          loading={saveProfile.isPending}
          error={saveProfile.isError ? errorMessage(saveProfile.error) : null}
          onSubmit={(input) => saveProfile.mutate(input, { onSuccess: next })}
        />
      ) : null}
      {step === 1 ? (
        <PreferencesForm
          initial={profile.preferences}
          submitLabel="Continuer"
          loading={savePrefs.isPending}
          error={savePrefs.isError ? errorMessage(savePrefs.error) : null}
          onSubmit={(input) => savePrefs.mutate(input, { onSuccess: next })}
        />
      ) : null}
      {step === 2 ? (
        <InterestsPicker
          initial={profile.interests}
          submitLabel="Continuer"
          loading={saveInterests.isPending}
          error={saveInterests.isError ? errorMessage(saveInterests.error) : null}
          onSubmit={(slugs) => saveInterests.mutate(slugs, { onSuccess: next })}
        />
      ) : null}
      {step === 3 ? (
        <View style={{ gap: 12 }}>
          <LocationControl hasLocation={profile.hasLocation} onDone={next} />
          <Button
            testID="location-skip"
            label={profile.hasLocation ? "Continuer" : "Plus tard"}
            variant="secondary"
            onPress={next}
          />
        </View>
      ) : null}
      {step === 4 ? <PhotosManager photos={profile.photos} /> : null}

      <View style={styles.footer}>
        {step > 0 ? (
          <Button label="Retour" variant="ghost" size="md" onPress={() => setStep((s) => s - 1)} />
        ) : (
          <View />
        )}
        <Button label="Se déconnecter" variant="ghost" size="md" onPress={() => void signOut()} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: 6 },
  bar: { flex: 1, height: 4 },
  footer: { flexDirection: "row", justifyContent: "space-between", marginTop: "auto" }
});
