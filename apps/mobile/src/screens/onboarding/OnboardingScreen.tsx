import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { getMyProfile, updateMyProfile, updatePreferences } from "../../api/profile";
import type { Gender, MyProfile, Photo } from "../../api/types";
import { setProfileCache, useAuth } from "../../hooks/useAuth";
import { useInterests } from "../../hooks/queries";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Icon, Screen, Text, TextField, showToast } from "../../ui";
import { shareApproximateLocation } from "../../utils/location";
import { firstNameProblem, parseBirthdate } from "../../utils/validation";
import { BirthdateInput, partsFromIso, type BirthdateParts } from "../profile/components/BirthdateInput";
import { PhotoGrid } from "../profile/components/PhotoGrid";
import { GenderPicker, InterestedInPicker, InterestPicker } from "../profile/components/Pickers";

const STEPS = ["identity", "gender", "photos", "about", "location"] as const;
type Step = (typeof STEPS)[number];

function initialStep(profile: MyProfile | null): Step {
  if (!profile?.completion.profile) return "identity";
  if (!profile.completion.preferences) return "gender";
  if (!profile.completion.photos) return "photos";
  return "about";
}

function defaultAgeRange(age: number) {
  return { minAge: Math.max(18, age - 7), maxAge: Math.min(99, age + 7) };
}

export function OnboardingScreen() {
  const { colors } = useTheme();
  const { profile, signOut, finishOnboarding } = useAuth();
  const interests = useInterests();
  const [step, setStep] = useState<Step>(() => initialStep(profile));
  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [birth, setBirth] = useState<BirthdateParts>(partsFromIso(profile?.birthdate ?? ""));
  const [gender, setGender] = useState<Gender | "">(profile?.gender ?? "");
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile?.preferences?.interestedIn ?? []);
  const [photos, setPhotos] = useState<Photo[]>(profile?.photos ?? []);
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(profile?.jobTitle ?? "");
  const [city, setCity] = useState(profile?.city ?? "");
  const [selectedInterests, setSelectedInterests] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const birthLocked = !!profile?.completion.profile;

  const index = STEPS.indexOf(step);
  const birthResult = useMemo(() => parseBirthdate(birth.day, birth.month, birth.year), [birth]);

  const saveProfile = () => {
    if (!birthResult.ok || !gender) throw new Error("incomplete");
    return updateMyProfile({
      firstName: firstName.trim(),
      birthdate: birthResult.iso,
      gender,
      bio,
      jobTitle,
      city,
      interests: selectedInterests
    });
  };

  const guarded = async (task: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await task();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const next = () => {
    switch (step) {
      case "identity": {
        const nameProblem = firstNameProblem(firstName);
        if (nameProblem) return setError(nameProblem);
        if (!birthResult.ok) return setError(birthResult.error);
        setError(null);
        return setStep("gender");
      }
      case "gender":
        if (!gender) return setError("Choisissez comment vous vous identifiez.");
        if (interestedIn.length === 0) return setError("Choisissez qui vous souhaitez rencontrer.");
        return guarded(async () => {
          await saveProfile();
          const age = birthResult.ok ? birthResult.age : 30;
          await updatePreferences({
            interestedIn,
            ...(profile?.preferences ? { minAge: profile.preferences.minAge, maxAge: profile.preferences.maxAge } : defaultAgeRange(age)),
            maxDistanceKm: profile?.preferences?.maxDistanceKm ?? 50
          });
          setStep("photos");
        });
      case "photos":
        if (photos.length === 0) return setError("Ajoutez au moins une photo.");
        setError(null);
        return setStep("about");
      case "about":
        return guarded(async () => {
          await saveProfile();
          setStep("location");
        });
      case "location":
        return finish();
    }
  };

  const finish = () =>
    guarded(async () => {
      setProfileCache(await getMyProfile());
      finishOnboarding();
    });

  const enableLocation = () =>
    guarded(async () => {
      const outcome = await shareApproximateLocation();
      if (outcome === "shared") {
        showToast("Localisation approximative enregistrée.", "success");
        await finish();
      } else {
        setError(
          outcome === "denied"
            ? "Autorisation refusée. Vous pourrez l'activer plus tard dans les réglages."
            : "Position indisponible pour le moment. Vous pourrez réessayer depuis les réglages."
        );
      }
    });

  const back = () => {
    setError(null);
    if (index > 0) setStep(STEPS[index - 1]);
  };

  return (
    <Screen
      edges={["top", "bottom"]}
      footer={
        step === "location" ? (
          <>
            <Button icon="navigate" label="Activer la localisation" loading={busy} onPress={enableLocation} testID="onboarding-location" />
            <Button disabled={busy} label="Plus tard" onPress={finish} testID="onboarding-skip-location" variant="ghost" />
          </>
        ) : (
          <Button label="Continuer" loading={busy} onPress={next} testID="onboarding-next" />
        )
      }
      scroll
      testID="onboarding-screen"
    >
      <View style={styles.top}>
        {index > 0 ? (
          <Button label="Retour" onPress={back} size="md" style={styles.backButton} variant="ghost" />
        ) : (
          <Button label="Se déconnecter" onPress={() => void signOut()} size="md" style={styles.backButton} variant="ghost" />
        )}
        <Text tone="subtle" variant="label">
          Étape {index + 1} sur {STEPS.length}
        </Text>
      </View>
      <View
        accessibilityLabel={`Progression ${index + 1} sur ${STEPS.length}`}
        style={[styles.progress, { backgroundColor: colors.surfaceMuted }]}
      >
        <View style={[styles.progressFill, { width: `${((index + 1) / STEPS.length) * 100}%`, backgroundColor: colors.primary }]} />
      </View>

      <View style={styles.body}>
        {step === "identity" ? (
          <>
            <Text variant="title">Faisons connaissance</Text>
            <Text tone="muted">Votre prénom et votre âge seront visibles sur votre profil.</Text>
            <TextField
              autoComplete="given-name"
              label="Prénom"
              onChangeText={setFirstName}
              testID="onboarding-first-name"
              value={firstName}
            />
            <BirthdateInput disabled={birthLocked} onChange={setBirth} value={birth} />
          </>
        ) : null}
        {step === "gender" ? (
          <>
            <Text variant="title">Ce que vous cherchez</Text>
            <GenderPicker onChange={setGender} value={gender} />
            <InterestedInPicker onChange={setInterestedIn} value={interestedIn} />
            <Text tone="subtle" variant="caption">
              Vous pourrez affiner l’âge et la distance dans vos préférences.
            </Text>
          </>
        ) : null}
        {step === "photos" ? (
          <>
            <Text variant="title">Vos plus belles photos</Text>
            <Text tone="muted">Ajoutez au moins une photo où l’on voit bien votre visage. La première sera votre photo principale.</Text>
            <PhotoGrid onChange={setPhotos} photos={photos} />
          </>
        ) : null}
        {step === "about" ? (
          <>
            <Text variant="title">Parlez un peu de vous</Text>
            <TextField
              label="Bio"
              maxLength={500}
              multiline
              onChangeText={setBio}
              placeholder="Ce qui vous fait sourire, votre dimanche idéal…"
              testID="onboarding-bio"
              value={bio}
            />
            <TextField label="Métier (facultatif)" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
            <TextField
              hint="Affichée telle quelle : un quartier ou une ville suffit."
              label="Ville (facultatif)"
              maxLength={60}
              onChangeText={setCity}
              value={city}
            />
            {interests.data ? <InterestPicker catalog={interests.data} onChange={setSelectedInterests} value={selectedInterests} /> : null}
          </>
        ) : null}
        {step === "location" ? (
          <View style={styles.locationBlock}>
            <View style={[styles.locationIcon, { backgroundColor: colors.primarySoft }]}>
              <Icon color={colors.primary} name="location" size={36} />
            </View>
            <Text center variant="title">
              Rencontrer près de chez vous
            </Text>
            <Text center tone="muted">
              Nous utilisons une position approximative (à environ 1 km près) uniquement pour calculer les distances. Votre position exacte
              n’est jamais enregistrée ni montrée.
            </Text>
          </View>
        ) : null}
        {error ? (
          <Text accessibilityLiveRegion="polite" testID="onboarding-error" tone="danger">
            {error}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingTop: spacing.xs },
  backButton: { paddingHorizontal: 0 },
  progress: { height: 6, borderRadius: radii.pill, overflow: "hidden", marginTop: spacing.xs },
  progressFill: { height: 6, borderRadius: radii.pill },
  body: { gap: spacing.lg, paddingTop: spacing.xl },
  locationBlock: { alignItems: "center", gap: spacing.md, paddingTop: spacing.xl },
  locationIcon: { width: 88, height: 88, borderRadius: 44, alignItems: "center", justifyContent: "center" }
});
