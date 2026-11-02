import React, { useState } from "react";
import { View } from "react-native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiErrorMessage } from "../../api/client";
import { photoApi, profileApi } from "../../api/endpoints";
import { useAuth } from "../../auth/AuthContext";
import {
  AboutFields,
  DEFAULT_PREFERENCES,
  IdentityFields,
  LocationFields,
  PreferenceFields,
  buildProfileInput,
  identityFromProfile,
  validateIdentity,
  type IdentityDraft,
  type LocationDraft
} from "../../components/ProfileForm";
import { PhotoManager } from "../../components/PhotoManager";
import { AppText, Button, Loading, Screen } from "../../components/ui";
import { useToast } from "../../components/Toast";
import { PROFILE_KEY, useProfileBundle } from "../../hooks";
import { spacing, useTheme } from "../../theme/theme";
import type { Preferences } from "../../api/types";

const STEPS = ["Vous", "Vos envies", "Où êtes-vous ?", "Votre histoire", "Vos photos"] as const;

export default function OnboardingScreen({ onFinish }: { onFinish: () => void }) {
  const bundle = useProfileBundle();
  if (bundle.isLoading) return <Loading label="Préparation…" />;
  return <Wizard bundle={bundle.data} onFinish={onFinish} />;
}

function Wizard({ bundle, onFinish }: { bundle: ReturnType<typeof useProfileBundle>["data"]; onFinish: () => void }) {
  const { colors } = useTheme();
  const toast = useToast();
  const { logout } = useAuth();
  const queryClient = useQueryClient();
  const profile = bundle?.profile ?? null;
  const photos = useQuery({ queryKey: ["photos"], queryFn: photoApi.list });

  // Resume where the user left: an existing profile with location only misses photos.
  const [step, setStep] = useState(profile && profile.latitude !== null && bundle?.preferences ? 4 : 0);
  const [identity, setIdentity] = useState<IdentityDraft>(identityFromProfile(profile));
  const [prefs, setPrefs] = useState<Preferences>(bundle?.preferences ?? DEFAULT_PREFERENCES);
  const [location, setLocation] = useState<LocationDraft>({
    latitude: profile?.latitude ?? null,
    longitude: profile?.longitude ?? null,
    city: profile?.city ?? ""
  });
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(profile?.interests.map((i) => i.id) ?? []);
  const [errors, setErrors] = useState<Partial<Record<"firstName" | "birthDate" | "gender", string>>>({});
  const [busy, setBusy] = useState(false);

  const next = async () => {
    if (step === 0) {
      const found = validateIdentity(identity);
      setErrors(found);
      if (Object.keys(found).length > 0) return;
    }
    if (step === 1 && prefs.interestedIn.length === 0) {
      toast.show("Choisissez au moins une option.", "error");
      return;
    }
    if (step === 2 && location.latitude === null) {
      toast.show("Indiquez votre position ou une ville pour continuer.", "error");
      return;
    }
    if (step === 3) {
      const input = buildProfileInput(identity, location, bio, interestIds, {
        showDistance: profile?.showDistance ?? true,
        discoverable: profile?.discoverable ?? true
      });
      if (!input) return;
      setBusy(true);
      try {
        await profileApi.save(input);
        await profileApi.savePreferences(prefs);
        await queryClient.invalidateQueries({ queryKey: PROFILE_KEY });
      } catch (e) {
        toast.show(apiErrorMessage(e), "error");
        setBusy(false);
        return;
      }
      setBusy(false);
    }
    setStep(step + 1);
  };

  const finish = () => {
    void queryClient.invalidateQueries({ queryKey: PROFILE_KEY });
    onFinish();
  };

  return (
    <Screen scroll>
      <View style={{ flexDirection: "row", gap: 6 }} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((label, i) => (
          <View key={label} style={{ flex: 1, height: 4, borderRadius: 2, backgroundColor: i <= step ? colors.primary : colors.border }} />
        ))}
      </View>
      <AppText variant="title">{STEPS[step]}</AppText>

      {step === 0 ? <IdentityFields draft={identity} onChange={setIdentity} errors={errors} /> : null}
      {step === 1 ? <PreferenceFields value={prefs} onChange={setPrefs} /> : null}
      {step === 2 ? <LocationFields value={location} onChange={setLocation} /> : null}
      {step === 3 ? <AboutFields bio={bio} onBio={setBio} selected={interestIds} onSelected={setInterestIds} /> : null}
      {step === 4 ? (
        <>
          <AppText muted>Ajoutez au moins une photo. Les profils avec plusieurs photos reçoivent plus de réponses.</AppText>
          <PhotoManager />
        </>
      ) : null}

      <View style={{ gap: spacing.sm, marginTop: spacing.md }}>
        {step < 4 ? (
          <Button label="Continuer" onPress={next} loading={busy} testID="onboarding-next" />
        ) : (
          <Button label="Terminer" onPress={finish} disabled={(photos.data?.items.length ?? 0) < 1} testID="onboarding-finish" />
        )}
        {step > 0 && step < 4 ? <Button label="Retour" variant="ghost" onPress={() => setStep(step - 1)} /> : null}
        {step === 0 ? <Button label="Se déconnecter" variant="ghost" onPress={() => void logout()} /> : null}
      </View>
    </Screen>
  );
}
