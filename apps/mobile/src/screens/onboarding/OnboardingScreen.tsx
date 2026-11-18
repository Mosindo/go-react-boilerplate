import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { errorMessage } from "../../api/errors";
import type { Gender, MyProfile, RelationshipGoal } from "../../api/types";
import { BirthdateField, type BirthdateParts } from "../../components/BirthdateField";
import { ChoiceGroup } from "../../components/ChoiceGroup";
import { InterestPicker } from "../../components/InterestPicker";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useFeedback } from "../../feedback";
import { useInterestsCatalog, useSaveInterests, useSaveLocation, useSavePreferences, useSaveProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { makeStyles, radii, spacing } from "../../theme";
import { Button, Card, Icon, IconButton, Screen, Stepper, Text, TextField } from "../../ui";
import { ageFromIso, toIsoDate } from "../../utils/dates";
import { genderLabels, genders, goalLabels, goals, interestedInLabels } from "../../utils/labels";
import { getApproximateLocation } from "../../utils/location";
import { validateFirstName } from "../../utils/validation";

const STEPS = ["Vous", "Vos envies", "Photos", "À propos", "Autour de vous"] as const;

function initialStep(profile: MyProfile): number {
  if (profile.missing.includes("profile")) {
    return 0;
  }
  if (profile.missing.includes("preferences")) {
    return 1;
  }
  if (profile.missing.includes("photos")) {
    return 2;
  }
  return 3;
}

export default function OnboardingScreen({ profile, onDone }: { profile: MyProfile; onDone: () => void }) {
  const styles = useStyles();
  const { signOut } = useAuth();
  const { toast } = useFeedback();
  const [step, setStep] = useState(() => initialStep(profile));
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();
  const saveInterests = useSaveInterests();
  const saveLocation = useSaveLocation();
  const catalog = useInterestsCatalog();

  const p = profile.profile;
  const [firstName, setFirstName] = useState(p?.firstName ?? "");
  const [birth, setBirth] = useState<BirthdateParts>(() => {
    const [year = "", month = "", day = ""] = (p?.birthdate ?? "").split("-");
    return { day, month, year };
  });
  const [gender, setGender] = useState<Gender[]>(p ? [p.gender] : []);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile.preferences?.interestedIn ?? []);
  const [minAge, setMinAge] = useState(profile.preferences?.minAge ?? 25);
  const [maxAge, setMaxAge] = useState(profile.preferences?.maxAge ?? 40);
  const [goal, setGoal] = useState<RelationshipGoal[]>([p?.relationshipGoal ?? "unsure"]);
  const [bio, setBio] = useState(p?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(p?.jobTitle ?? "");
  const [interests, setInterests] = useState<string[]>(profile.interests.map((i) => i.slug));
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});
  const [locating, setLocating] = useState(false);

  useEffect(() => {
    // Sensible default age range around the member’s own age.
    if (!profile.preferences && p?.birthdate) {
      const age = ageFromIso(p.birthdate);
      setMinAge(Math.max(18, age - 6));
      setMaxAge(Math.min(99, age + 8));
    }
  }, [p?.birthdate, profile.preferences]);

  const profileInput = () => ({
    firstName: firstName.trim(),
    birthdate: toIsoDate(birth.day, birth.month, birth.year) ?? "",
    gender: gender[0],
    bio: bio.trim(),
    jobTitle: jobTitle.trim(),
    city: p?.city ?? "",
    heightCm: p?.heightCm ?? null,
    relationshipGoal: goal[0] ?? "unsure"
  });

  const submitBasics = async () => {
    const birthdate = toIsoDate(birth.day, birth.month, birth.year);
    const next: Record<string, string | undefined> = {
      firstName: validateFirstName(firstName) ?? undefined,
      birthdate: !birthdate ? "Date invalide." : ageFromIso(birthdate) < 18 ? "Lueur est réservée aux personnes majeures." : undefined,
      gender: gender.length ? undefined : "Choisissez une option."
    };
    setErrors(next);
    if (Object.values(next).some(Boolean)) {
      return;
    }
    try {
      await saveProfile.mutateAsync(profileInput());
      setStep(1);
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    }
  };

  const submitPreferences = async () => {
    if (interestedIn.length === 0) {
      setErrors({ interestedIn: "Choisissez au moins une option." });
      return;
    }
    setErrors({});
    try {
      await savePreferences.mutateAsync({ interestedIn, minAge, maxAge, maxDistanceKm: profile.preferences?.maxDistanceKm ?? 50 });
      setStep(2);
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    }
  };

  const submitAbout = async () => {
    try {
      await saveProfile.mutateAsync(profileInput());
      await saveInterests.mutateAsync(interests);
      setStep(4);
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    }
  };

  const locate = async () => {
    setLocating(true);
    const result = await getApproximateLocation();
    setLocating(false);
    if (result.status === "ok") {
      try {
        await saveLocation.mutateAsync(result);
        toast("Position enregistrée (approximative)", "success");
        onDone();
      } catch (e) {
        toast(errorMessage(e), "error");
      }
    } else {
      toast(
        result.status === "denied"
          ? "Sans localisation, nous vous montrerons des profils sans filtre de distance."
          : "Position indisponible pour le moment.",
        "info"
      );
    }
  };

  const content = () => {
    switch (step) {
      case 0:
        return (
          <>
            <Title title="Faisons connaissance" subtitle="Ces informations apparaissent sur votre profil." />
            <TextField
              autoCapitalize="words"
              autoComplete="given-name"
              error={errors.firstName}
              label="Prénom"
              maxLength={30}
              onChangeText={setFirstName}
              testID="onboarding-first-name"
              value={firstName}
            />
            <View style={{ gap: spacing.xs }}>
              <Text color="textMuted" variant="label">
                Date de naissance
              </Text>
              <BirthdateField error={errors.birthdate} onChange={setBirth} value={birth} />
            </View>
            <ChoiceGroup
              label="Je suis"
              labels={genderLabels}
              onChange={setGender}
              options={genders}
              testIDPrefix="gender"
              value={gender}
            />
            {errors.gender ? (
              <Text color="danger" variant="caption">
                {errors.gender}
              </Text>
            ) : null}
            <Button label="Continuer" loading={saveProfile.isPending} onPress={submitBasics} testID="onboarding-next" />
          </>
        );
      case 1:
        return (
          <>
            <Title title="Qui aimeriez-vous rencontrer ?" subtitle="Vous pourrez changer d’avis à tout moment." />
            <ChoiceGroup
              label="Je souhaite rencontrer"
              labels={interestedInLabels}
              multiple
              onChange={setInterestedIn}
              options={genders}
              testIDPrefix="interested"
              value={interestedIn}
            />
            {errors.interestedIn ? (
              <Text color="danger" variant="caption">
                {errors.interestedIn}
              </Text>
            ) : null}
            <Card>
              <Text variant="bodyStrong">Tranche d’âge</Text>
              <Stepper label="À partir de" max={maxAge} min={18} onChange={setMinAge} suffix=" ans" value={minAge} />
              <Stepper label="Jusqu’à" max={99} min={minAge} onChange={setMaxAge} suffix=" ans" value={maxAge} />
            </Card>
            <ChoiceGroup
              label="Je cherche plutôt"
              labels={goalLabels}
              onChange={setGoal}
              options={goals}
              testIDPrefix="goal"
              value={goal}
            />
            <Button label="Continuer" loading={savePreferences.isPending} onPress={submitPreferences} testID="onboarding-next" />
          </>
        );
      case 2:
        return (
          <>
            <Title title="Montrez-vous" subtitle="Au moins une photo où l’on voit bien votre visage. Jusqu’à 6." />
            <PhotoGrid photos={profile.photos} />
            <Card>
              <Text color="textMuted" variant="caption">
                Les photos sont recadrées et compressées sur votre appareil, puis débarrassées de leurs métadonnées (dont la position GPS).
              </Text>
            </Card>
            <Button disabled={profile.photos.length === 0} label="Continuer" onPress={() => setStep(3)} testID="onboarding-next" />
          </>
        );
      case 3:
        return (
          <>
            <Title title="Un mot sur vous" subtitle="Facultatif, mais les profils complets obtiennent bien plus de matchs." />
            <TextField
              hint={`${bio.length}/500`}
              label="Bio"
              maxLength={500}
              multiline
              onChangeText={setBio}
              placeholder="Ce qui vous fait vibrer, votre dimanche idéal…"
              testID="onboarding-bio"
              value={bio}
            />
            <TextField label="Métier" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
            <Text variant="heading">Centres d’intérêt</Text>
            <InterestPicker catalog={catalog.data} onChange={setInterests} selected={interests} />
            {errors.form ? (
              <Text color="danger" variant="caption">
                {errors.form}
              </Text>
            ) : null}
            <Button
              label="Continuer"
              loading={saveProfile.isPending || saveInterests.isPending}
              onPress={submitAbout}
              testID="onboarding-next"
            />
          </>
        );
      default:
        return (
          <>
            <Title title="Qui est près de vous ?" subtitle="Votre position sert uniquement à calculer une distance approximative." />
            <Card style={styles.privacy}>
              <Icon color="accent" name="shield-checkmark-outline" size={28} />
              <View style={styles.flex}>
                <Text variant="bodyStrong">Votre adresse reste privée</Text>
                <Text color="textMuted" variant="caption">
                  Nous arrondissons votre position à environ 1 km et n’affichons jamais que la distance. Vous pouvez la masquer ou la
                  supprimer.
                </Text>
              </View>
            </Card>
            <Button
              icon="navigate-outline"
              label="Utiliser ma position"
              loading={locating || saveLocation.isPending}
              onPress={locate}
              testID="onboarding-locate"
            />
            <Button label="Plus tard" onPress={onDone} testID="onboarding-skip-location" variant="ghost" />
          </>
        );
    }
  };

  return (
    <Screen keyboard scroll testID="onboarding-screen">
      <View style={styles.top}>
        {step > 0 ? (
          <IconButton icon="chevron-back" label="Étape précédente" onPress={() => setStep((s) => s - 1)} />
        ) : (
          <View style={styles.spacer} />
        )}
        <View style={styles.progress}>
          {STEPS.map((label, i) => (
            <View key={label} style={[styles.dot, i <= step ? styles.dotActive : null]} />
          ))}
        </View>
        <IconButton icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} />
      </View>
      <Text color="accent" variant="overline">
        Étape {step + 1} sur {STEPS.length} · {STEPS[step]}
      </Text>
      {content()}
      {errors.form && step !== 3 ? (
        <Text color="danger" variant="caption">
          {errors.form}
        </Text>
      ) : null}
    </Screen>
  );
}

function Title({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <View style={{ gap: spacing.xs }}>
      <Text accessibilityRole="header" variant="title">
        {title}
      </Text>
      <Text color="textMuted">{subtitle}</Text>
    </View>
  );
}

const useStyles = makeStyles((c) => ({
  top: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  spacer: { width: 44 },
  progress: { flexDirection: "row", gap: 6, flex: 1, justifyContent: "center" },
  dot: { height: 6, flex: 1, maxWidth: 44, borderRadius: radii.pill, backgroundColor: c.border },
  dotActive: { backgroundColor: c.accent },
  privacy: { flexDirection: "row", gap: spacing.md, alignItems: "flex-start" },
  flex: { flex: 1, gap: 4 }
}));
