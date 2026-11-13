import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, OwnProfile, RelationshipGoal } from "../../api/types";
import { profileApi } from "../../api/profile";
import { useOnboardingGate } from "../../navigation/RootNavigator";
import type { RootScreenProps } from "../../navigation/types";
import { useMyProfile, useProfileMutation, useUpdateProfile } from "../../state/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, ErrorState, Header, LoadingState, ProgressBar, Screen, Text, TextField } from "../../ui";
import { describeError } from "../../utils/errors";
import { ageFromIso, genderLabels, goalLabels, interestedInLabels, toIsoDate } from "../../utils/format";
import { InterestPicker } from "../profile/InterestPicker";
import { LocationSection } from "../profile/LocationSection";
import { PhotoManager } from "../profile/PhotoManager";

const STEPS = ["identity", "orientation", "photos", "location", "about"] as const;
type Step = (typeof STEPS)[number];

function firstStep(profile: OwnProfile): number {
  const m = profile.missing;
  if (m.includes("firstName") || m.includes("birthdate")) return 0;
  if (m.includes("gender") || m.includes("interestedIn")) return 1;
  if (m.includes("photos")) return 2;
  if (m.includes("location")) return 3;
  return 4;
}

export default function OnboardingScreen({ navigation }: RootScreenProps<"Onboarding" | "CompleteProfile">) {
  const profile = useMyProfile();
  if (profile.isLoading) {
    return <LoadingState />;
  }
  if (!profile.data) {
    return (
      <Screen>
        <ErrorState message={describeError(profile.error)} onAction={() => void profile.refetch()} />
      </Screen>
    );
  }
  return <Wizard profile={profile.data} canGoBack={navigation.canGoBack()} onExit={() => navigation.goBack()} />;
}

function Wizard({ profile, canGoBack, onExit }: { profile: OwnProfile; canGoBack: boolean; onExit: () => void }) {
  const { spacing } = useTheme();
  const gate = useOnboardingGate();
  const [stepIndex, setStepIndex] = useState(() => firstStep(profile));
  const step: Step = STEPS[stepIndex];
  const next = () => setStepIndex((i) => Math.min(i + 1, STEPS.length - 1));
  const back = () => setStepIndex((i) => Math.max(i - 1, 0));

  const finish = () => {
    if (canGoBack) {
      onExit();
    } else {
      gate.finish();
    }
  };

  return (
    <Screen scroll testID={`onboarding-${step}`}>
      <Header back={canGoBack && stepIndex === 0} />
      <View style={{ gap: spacing.lg }}>
        <ProgressBar value={(stepIndex + 1) / STEPS.length} />
        {stepIndex > 0 ? (
          <Button
            label="Étape précédente"
            variant="ghost"
            size="md"
            icon="arrow-back"
            onPress={back}
            style={styles.backBtn}
          />
        ) : null}
        {step === "identity" ? <IdentityStep profile={profile} onNext={next} /> : null}
        {step === "orientation" ? <OrientationStep profile={profile} onNext={next} /> : null}
        {step === "photos" ? <PhotosStep profile={profile} onNext={next} /> : null}
        {step === "location" ? <LocationStep profile={profile} onNext={next} /> : null}
        {step === "about" ? <AboutStep profile={profile} onFinish={finish} /> : null}
      </View>
    </Screen>
  );
}

function StepTitle({ title, subtitle }: { title: string; subtitle?: string }) {
  return (
    <View style={{ gap: 6 }}>
      <Text variant="title" accessibilityRole="header">
        {title}
      </Text>
      {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
    </View>
  );
}

function IdentityStep({ profile, onNext }: { profile: OwnProfile; onNext: () => void }) {
  const { spacing } = useTheme();
  const [y, m, d] = profile.birthdate?.split("-") ?? ["", "", ""];
  const [firstName, setFirstName] = useState(profile.firstName);
  const [day, setDay] = useState(d ?? "");
  const [month, setMonth] = useState(m ?? "");
  const [year, setYear] = useState(y ?? "");
  const [localError, setLocalError] = useState<string | null>(null);
  const update = useUpdateProfile();

  const submit = () => {
    const iso = toIsoDate(day, month, year);
    if (!firstName.trim()) {
      setLocalError("Indiquez votre prénom.");
      return;
    }
    if (!iso) {
      setLocalError("Date de naissance invalide.");
      return;
    }
    if (ageFromIso(iso) < 18) {
      setLocalError("Vous devez avoir au moins 18 ans pour utiliser Aimant.");
      return;
    }
    setLocalError(null);
    update.mutate({ firstName: firstName.trim(), birthdate: iso }, { onSuccess: onNext });
  };

  return (
    <View style={{ gap: spacing.lg }}>
      <StepTitle
        title="Faisons connaissance"
        subtitle="Votre prénom et votre âge seront visibles. Votre date de naissance, jamais."
      />
      <TextField
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        autoComplete="given-name"
        maxLength={40}
        testID="onboarding-first-name"
      />
      <View style={{ gap: 6 }}>
        <Text variant="label" weight="medium" tone="muted">
          Date de naissance
        </Text>
        <View style={styles.dateRow}>
          <View style={styles.dateShort}>
            <TextField
              placeholder="JJ"
              value={day}
              onChangeText={(v) => setDay(v.replace(/\D/g, "").slice(0, 2))}
              keyboardType="number-pad"
              accessibilityLabel="Jour"
              testID="onboarding-birth-day"
            />
          </View>
          <View style={styles.dateShort}>
            <TextField
              placeholder="MM"
              value={month}
              onChangeText={(v) => setMonth(v.replace(/\D/g, "").slice(0, 2))}
              keyboardType="number-pad"
              accessibilityLabel="Mois"
              testID="onboarding-birth-month"
            />
          </View>
          <View style={styles.dateLong}>
            <TextField
              placeholder="AAAA"
              value={year}
              onChangeText={(v) => setYear(v.replace(/\D/g, "").slice(0, 4))}
              keyboardType="number-pad"
              accessibilityLabel="Année"
              testID="onboarding-birth-year"
            />
          </View>
        </View>
      </View>
      {localError || update.error ? <Text tone="danger">{localError ?? describeError(update.error)}</Text> : null}
      <Button label="Continuer" onPress={submit} loading={update.isPending} testID="onboarding-next" />
    </View>
  );
}

function OrientationStep({ profile, onNext }: { profile: OwnProfile; onNext: () => void }) {
  const { spacing } = useTheme();
  const [gender, setGender] = useState<Gender | null>(profile.gender);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile.preferences.interestedIn);
  const update = useUpdateProfile();
  const prefs = useProfileMutation((values: Gender[]) => profileApi.updatePreferences({ interestedIn: values }));
  const genders = Object.keys(genderLabels) as Gender[];

  const submit = async () => {
    if (!gender || interestedIn.length === 0) return;
    try {
      await update.mutateAsync({ gender });
      await prefs.mutateAsync(interestedIn);
      onNext();
    } catch {
      // Error rendered below.
    }
  };

  return (
    <View style={{ gap: spacing.lg }}>
      <StepTitle title="Vous êtes…" />
      <View style={styles.chips}>
        {genders.map((g) => (
          <Chip
            key={g}
            label={genderLabels[g]}
            selected={gender === g}
            onPress={() => setGender(g)}
            testID={`gender-${g}`}
          />
        ))}
      </View>
      <StepTitle title="Vous souhaitez rencontrer…" subtitle="Plusieurs choix possibles. Modifiable à tout moment." />
      <View style={styles.chips}>
        {genders.map((g) => (
          <Chip
            key={g}
            label={interestedInLabels[g]}
            selected={interestedIn.includes(g)}
            onPress={() => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]))}
            testID={`interested-${g}`}
          />
        ))}
      </View>
      {update.error || prefs.error ? <Text tone="danger">{describeError(update.error ?? prefs.error)}</Text> : null}
      <Button
        label="Continuer"
        onPress={submit}
        disabled={!gender || interestedIn.length === 0}
        loading={update.isPending || prefs.isPending}
        testID="onboarding-next"
      />
    </View>
  );
}

function PhotosStep({ profile, onNext }: { profile: OwnProfile; onNext: () => void }) {
  const { spacing } = useTheme();
  return (
    <View style={{ gap: spacing.lg }}>
      <StepTitle
        title="Montrez-vous"
        subtitle="Ajoutez au moins une photo où l'on voit bien votre visage. Les métadonnées (dont la position GPS) sont supprimées."
      />
      <PhotoManager photos={profile.photos} />
      <Button label="Continuer" onPress={onNext} disabled={profile.photos.length === 0} testID="onboarding-next" />
    </View>
  );
}

function LocationStep({ profile, onNext }: { profile: OwnProfile; onNext: () => void }) {
  const { spacing } = useTheme();
  return (
    <View style={{ gap: spacing.lg }}>
      <StepTitle
        title="Où êtes-vous ?"
        subtitle="Nous l'utilisons pour vous proposer des personnes proches. Les autres ne voient qu'une distance approximative, jamais votre position."
      />
      <LocationSection city={profile.city} hasLocation={profile.hasLocation} />
      <Button label="Continuer" onPress={onNext} disabled={!profile.hasLocation} testID="onboarding-next" />
    </View>
  );
}

function AboutStep({ profile, onFinish }: { profile: OwnProfile; onFinish: () => void }) {
  const { spacing } = useTheme();
  const [bio, setBio] = useState(profile.bio);
  const [goal, setGoal] = useState<RelationshipGoal | null>(profile.relationshipGoal);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const update = useUpdateProfile();
  const goals = useMemo(() => Object.keys(goalLabels) as RelationshipGoal[], []);

  const submit = () =>
    update.mutate(
      { bio: bio.trim(), interests, relationshipGoal: goal ?? "" },
      {
        onSuccess: (updated) => {
          if (updated.isComplete) onFinish();
        }
      }
    );

  const stillMissing = update.data && !update.data.isComplete ? update.data.missing : null;

  return (
    <View style={{ gap: spacing.lg }}>
      <StepTitle
        title="Encore un mot sur vous"
        subtitle="Facultatif, mais les profils complets reçoivent plus de réponses."
      />
      <TextField
        label="Bio"
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
        placeholder="Ce qui vous fait vibrer, ce que vous cherchez…"
        hint={`${bio.length}/500`}
        testID="onboarding-bio"
      />
      <Text variant="label" weight="medium" tone="muted">
        Ce que vous recherchez
      </Text>
      <View style={styles.chips}>
        {goals.map((g) => (
          <Chip key={g} label={goalLabels[g]} selected={goal === g} onPress={() => setGoal(goal === g ? null : g)} />
        ))}
      </View>
      <Text variant="label" weight="medium" tone="muted">
        Centres d’intérêt
      </Text>
      <InterestPicker selected={interests} onChange={setInterests} />
      {update.error ? <Text tone="danger">{describeError(update.error)}</Text> : null}
      {stillMissing ? (
        <Text tone="danger">Il manque encore : {stillMissing.join(", ")}. Revenez aux étapes précédentes.</Text>
      ) : null}
      <Button label="Découvrir des profils" onPress={submit} loading={update.isPending} testID="onboarding-finish" />
    </View>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  dateRow: { flexDirection: "row", gap: 10 },
  dateShort: { width: 76 },
  dateLong: { width: 104 },
  backBtn: { alignSelf: "flex-start", paddingHorizontal: 0 }
});
