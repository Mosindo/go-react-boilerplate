import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, MeProfile } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { checkBirthdate, isValidFirstName } from "../../lib/validation";
import { useSession } from "../../session/SessionProvider";
import { radii, spacing, useTheme } from "../../theme";
import { Button, InlineError, Screen, Text, TextField, toast } from "../../ui";
import { BirthdateField, GenderPicker, InterestedInPicker, InterestPicker, type BirthdateParts } from "../profile/fields";
import { profileToUpdate, useInterests, useUpdateLocation, useUpdatePreferences, useUpdateProfile } from "../profile/hooks";
import { currentApproximateLocation } from "../profile/media";
import { PhotoGrid } from "../profile/PhotoGrid";

const STEPS = ["name", "birthdate", "gender", "seeking", "photos", "about", "location"] as const;
type Step = (typeof STEPS)[number];

function initialStep(me: MeProfile): number {
  const { profile } = me;
  if (!profile.firstName) return 0;
  if (!profile.birthdate) return 1;
  if (!profile.gender) return 2;
  if (profile.photos.length === 0) return 4;
  return 5;
}

export function OnboardingScreen({ me, onFinish }: { me: MeProfile; onFinish: () => void }) {
  const { colors } = useTheme();
  const { signOut } = useSession();
  const [index, setIndex] = useState(() => initialStep(me));
  const step: Step = STEPS[index];
  const [firstName, setFirstName] = useState(me.profile.firstName);
  const [birth, setBirth] = useState<BirthdateParts>(() => {
    const [year = "", month = "", day = ""] = (me.profile.birthdate ?? "").split("-");
    return { day, month, year };
  });
  const [gender, setGender] = useState<Gender | "">(me.profile.gender ?? "");
  const [seeking, setSeeking] = useState<Gender[]>(me.preferences.interestedIn);
  const [interestIds, setInterestIds] = useState<number[]>(me.profile.interests.map((i) => i.id));
  const [bio, setBio] = useState(me.profile.bio);
  const [city, setCity] = useState(me.profile.city);
  const [error, setError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);

  const interests = useInterests();
  const updateProfile = useUpdateProfile();
  const updatePreferences = useUpdatePreferences();
  const updateLocation = useUpdateLocation();
  const busy = updateProfile.isPending || updatePreferences.isPending || updateLocation.isPending;

  const birthCheck = useMemo(() => checkBirthdate(birth.day, birth.month, birth.year), [birth]);
  const birthLocked = Boolean(me.profile.birthdate);

  const next = () => {
    setError(null);
    setIndex((i) => Math.min(i + 1, STEPS.length - 1));
  };

  const saveIdentity = async () => {
    if (!birthCheck.ok) {
      setError(birthCheck.error);
      return;
    }
    if (!gender) {
      setError("Choisissez une option.");
      return;
    }
    try {
      await updateProfile.mutateAsync({
        ...profileToUpdate(me.profile),
        firstName: firstName.trim(),
        birthdate: birthCheck.iso,
        gender
      });
      next();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const saveSeeking = async () => {
    const age = birthCheck.ok ? birthCheck.age : 30;
    try {
      await updatePreferences.mutateAsync({
        interestedIn: seeking,
        minAge: Math.max(18, age - 8),
        maxAge: Math.min(99, age + 8),
        maxDistanceKm: me.preferences.maxDistanceKm
      });
      next();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const saveAbout = async (current: MeProfile) => {
    try {
      await updateProfile.mutateAsync({ ...profileToUpdate(current.profile), bio, interestIds });
      next();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const locate = async () => {
    setLocating(true);
    setError(null);
    const result = await currentApproximateLocation();
    setLocating(false);
    if ("error" in result) {
      setError(result.error);
      return;
    }
    try {
      await updateLocation.mutateAsync({ ...result, city: result.city || city });
      toast("Position approximative enregistrée", "success");
      onFinish();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const finishWithCity = async (current: MeProfile) => {
    try {
      if (city.trim() !== current.profile.city) {
        await updateProfile.mutateAsync({ ...profileToUpdate(current.profile), city: city.trim() });
      }
      onFinish();
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const content: Record<Step, { title: string; subtitle?: string; body: React.ReactNode; action: React.ReactNode }> = {
    name: {
      title: "Comment vous appelez-vous ?",
      subtitle: "Votre prénom sera visible sur votre profil.",
      body: (
        <TextField
          label="Prénom"
          value={firstName}
          onChangeText={setFirstName}
          autoComplete="given-name"
          textContentType="givenName"
          maxLength={40}
          testID="onboarding-name"
        />
      ),
      action: <Button label="Continuer" disabled={!isValidFirstName(firstName)} onPress={next} testID="onboarding-next" />
    },
    birthdate: {
      title: "Votre date de naissance",
      subtitle: birthLocked ? "Elle est déjà enregistrée." : "Seul votre âge est affiché. Elle ne pourra plus être modifiée.",
      body: (
        <View style={{ gap: spacing.sm }}>
          <BirthdateField
            value={birth}
            onChange={birthLocked ? () => undefined : setBirth}
            error={birth.year.length === 4 && !birthCheck.ok ? birthCheck.error : null}
          />
          {birthCheck.ok ? <Text color="textMuted">Vous avez {birthCheck.age} ans.</Text> : null}
        </View>
      ),
      action: <Button label="Continuer" disabled={!birthCheck.ok} onPress={next} testID="onboarding-next" />
    },
    gender: {
      title: "Vous êtes…",
      body: <GenderPicker value={gender} onChange={setGender} />,
      action: <Button label="Continuer" disabled={!gender} loading={busy} onPress={saveIdentity} testID="onboarding-next" />
    },
    seeking: {
      title: "Vous aimeriez rencontrer…",
      subtitle: "Plusieurs choix possibles. Modifiable à tout moment.",
      body: <InterestedInPicker value={seeking} onChange={setSeeking} />,
      action: <Button label="Continuer" loading={busy} onPress={saveSeeking} testID="onboarding-next" />
    },
    photos: {
      title: "Ajoutez vos photos",
      subtitle: "Au moins une photo, où l'on voit bien votre visage.",
      body: <PhotoGrid photos={me.profile.photos} />,
      action: <Button label="Continuer" disabled={me.profile.photos.length === 0} onPress={next} testID="onboarding-next" />
    },
    about: {
      title: "Parlez un peu de vous",
      subtitle: "Facultatif, mais les profils complets font plus de rencontres.",
      body: (
        <View style={{ gap: spacing.lg }}>
          <TextField
            label="Bio"
            value={bio}
            onChangeText={setBio}
            multiline
            maxLength={500}
            placeholder="Ce qui vous fait vibrer, ce que vous cherchez…"
            testID="onboarding-bio"
          />
          {interests.data ? <InterestPicker all={interests.data} selected={interestIds} onChange={setInterestIds} /> : null}
        </View>
      ),
      action: <Button label="Continuer" loading={busy} onPress={() => saveAbout(me)} testID="onboarding-next" />
    },
    location: {
      title: "Où êtes-vous ?",
      subtitle:
        "Votre position sert uniquement à calculer des distances approximatives. Elle est arrondie (~1 km) et jamais montrée aux autres.",
      body: (
        <View style={{ gap: spacing.md }}>
          <Button
            label="Utiliser ma position"
            icon="navigate-outline"
            variant="secondary"
            loading={locating}
            onPress={locate}
            testID="onboarding-locate"
          />
          <TextField
            label="Ou indiquez votre ville"
            value={city}
            onChangeText={setCity}
            maxLength={80}
            autoComplete="postal-address-locality"
            testID="onboarding-city"
          />
        </View>
      ),
      action: <Button label="Terminer" loading={busy} onPress={() => finishWithCity(me)} testID="onboarding-finish" />
    }
  };

  const current = content[step];
  return (
    <Screen
      scroll
      testID={`onboarding-step-${step}`}
      edges={["top", "bottom"]}
      footer={
        <View style={{ gap: spacing.sm }}>
          <InlineError message={error} />
          {current.action}
        </View>
      }
    >
      <View style={styles.top}>
        <View style={[styles.progress, { backgroundColor: colors.surfaceAlt }]}>
          <View style={[styles.bar, { backgroundColor: colors.primary, width: `${((index + 1) / STEPS.length) * 100}%` }]} />
        </View>
        <View style={styles.nav}>
          {index > 0 && !(step === "seeking" || step === "photos") ? (
            <Button label="Retour" variant="ghost" compact onPress={() => setIndex((i) => Math.max(0, i - 1))} />
          ) : (
            <View />
          )}
          <Button label="Se déconnecter" variant="ghost" compact onPress={() => void signOut()} />
        </View>
      </View>
      <Text variant="title" accessibilityRole="header">
        {current.title}
      </Text>
      {current.subtitle ? (
        <Text color="textMuted" style={styles.subtitle}>
          {current.subtitle}
        </Text>
      ) : null}
      <View style={styles.body}>{current.body}</View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: { paddingTop: spacing.sm, marginBottom: spacing.lg, gap: spacing.xs },
  progress: { height: 6, borderRadius: radii.pill, overflow: "hidden" },
  bar: { height: "100%", borderRadius: radii.pill },
  nav: { flexDirection: "row", justifyContent: "space-between" },
  subtitle: { marginTop: spacing.xs },
  body: { marginTop: spacing.xl }
});
