import React, { useState } from "react";
import { View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { saveProfile, savePreferences } from "../api/profile";
import type { OwnProfile } from "../api/types";
import { useAuth } from "../auth/AuthProvider";
import { PhotosManager } from "../components/PhotosManager";
import { DEFAULT_PREFERENCES, PreferencesForm } from "../components/PreferencesForm";
import { ProfileForm } from "../components/ProfileForm";
import { Button, Screen, Text } from "../ui";
import { spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";

type Props = { profile: OwnProfile | null };

/** Which step to show: profile -> photos (at least one, then "Continuer") -> preferences. */
export function onboardingStep(profile: OwnProfile | null, photosConfirmed: boolean): 1 | 2 | 3 {
  if (!profile) return 1;
  if (profile.photos.length === 0 || !photosConfirmed) return 2;
  return 3;
}

export default function OnboardingScreen({ profile }: Props) {
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const { colors } = useTheme();
  const [photosConfirmed, setPhotosConfirmed] = useState(!!profile && profile.photos.length > 0);
  const step = onboardingStep(profile, photosConfirmed);

  return (
    <Screen scroll edges={["top", "bottom"]} testID={`onboarding-step-${step}`}>
      <View style={{ flexDirection: "row", gap: spacing.xs }} accessibilityLabel={`Étape ${step} sur 3`}>
        {[1, 2, 3].map((n) => (
          <View
            key={n}
            style={{
              flex: 1,
              height: 4,
              borderRadius: 2,
              backgroundColor: n <= step ? colors.accent : colors.border
            }}
          />
        ))}
      </View>

      {step === 1 ? (
        <>
          <Text variant="title">Faisons connaissance</Text>
          <Text tone="muted">Ces informations apparaîtront sur votre profil.</Text>
          <ProfileForm
            submitLabel="Continuer"
            onSubmit={async (input) => {
              const saved = await saveProfile(input);
              queryClient.setQueryData(["profile"], saved);
            }}
          />
        </>
      ) : null}

      {step === 2 && profile ? (
        <>
          <Text variant="title">Vos plus belles photos</Text>
          <Text tone="muted">
            Ajoutez au moins une photo (6 maximum). La première est votre photo principale.
          </Text>
          <PhotosManager photos={profile.photos} />
          <Button
            label="Continuer"
            onPress={() => setPhotosConfirmed(true)}
            disabled={profile.photos.length === 0}
            testID="photos-continue"
          />
        </>
      ) : null}

      {step === 3 && profile ? (
        <>
          <Text variant="title">Qui souhaitez-vous rencontrer ?</Text>
          <PreferencesForm
            initial={{ ...DEFAULT_PREFERENCES }}
            submitLabel="Découvrir des profils"
            onSubmit={async (prefs) => {
              const saved = await savePreferences(prefs);
              queryClient.setQueryData<OwnProfile | null>(["profile"], (current) =>
                current ? { ...current, preferences: saved } : current
              );
            }}
          />
        </>
      ) : null}

      <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
    </Screen>
  );
}
