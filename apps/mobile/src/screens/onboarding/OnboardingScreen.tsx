import React, { useEffect, useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, MyProfile } from "../../api/types";
import { AudienceField, BirthdateField, GenderField, InterestsField, RangeFields } from "../../components/ProfileFields";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useInterests, useUpdateLocation, useUpdatePreferences, useUpdateProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Card, Input, Text, colors, radii, spacing } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { ageFromIso, toIsoDate } from "../../utils/format";
import { readApproximateLocation } from "../../utils/media";

const STEPS = ["basics", "looking", "photos", "about", "location"] as const;
type Step = (typeof STEPS)[number];

function firstStep(profile: MyProfile): Step {
  const m = profile.missing;
  if (m.includes("firstName") || m.includes("birthdate") || m.includes("gender")) {
    return "basics";
  }
  if (m.includes("interestedIn")) {
    return "looking";
  }
  if (m.includes("photo")) {
    return "photos";
  }
  return profile.bio ? "location" : "about";
}

type Props = { profile: MyProfile; onFinish: () => void };

export default function OnboardingScreen({ onFinish, profile }: Props) {
  const { signOut } = useAuth();
  const [step, setStep] = useState<Step>(() => firstStep(profile));
  const index = STEPS.indexOf(step);
  const next = () => (index < STEPS.length - 1 ? setStep(STEPS[index + 1]) : onFinish());
  const back = index > 0 ? () => setStep(STEPS[index - 1]) : undefined;

  return (
    <Screen
      footer={
        <View style={styles.progress}>
          {STEPS.map((s, i) => (
            <View key={s} style={[styles.progressDot, i <= index ? styles.progressDotActive : null]} />
          ))}
        </View>
      }
      testID={`onboarding-${step}`}
    >
      {step === "basics" ? <BasicsStep onDone={next} profile={profile} /> : null}
      {step === "looking" ? <LookingStep onBack={back} onDone={next} profile={profile} /> : null}
      {step === "photos" ? <PhotosStep onBack={back} onDone={next} profile={profile} /> : null}
      {step === "about" ? <AboutStep onBack={back} onDone={next} profile={profile} /> : null}
      {step === "location" ? <LocationStep onBack={back} onDone={onFinish} profile={profile} /> : null}
      {index === 0 ? <Button label="Se déconnecter" onPress={() => void signOut()} size="sm" variant="ghost" /> : null}
    </Screen>
  );
}

type StepProps = { profile: MyProfile; onDone: () => void; onBack?: () => void };

function Nav({
  onBack,
  onNext,
  disabled,
  loading,
  label = "Continuer"
}: {
  onBack?: () => void;
  onNext: () => void;
  disabled?: boolean;
  loading?: boolean;
  label?: string;
}) {
  return (
    <View style={styles.nav}>
      {onBack ? <Button label="Retour" onPress={onBack} style={styles.navBack} variant="secondary" /> : null}
      <Button disabled={disabled} label={label} loading={loading} onPress={onNext} style={styles.navNext} testID="onboarding-next" />
    </View>
  );
}

function BasicsStep({ onDone, profile }: StepProps) {
  const update = useUpdateProfile();
  const [name, setName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender | "">(profile.gender);
  const [birth, setBirth] = useState(() => {
    const [year = "", month = "", day = ""] = (profile.birthdate ?? "").split("-");
    return { day, month, year };
  });
  const iso = toIsoDate(birth.day, birth.month, birth.year);
  const dateError =
    birth.year.length === 4 && !iso ? "Date invalide." : iso && ageFromIso(iso) < 18 ? "Hearth est réservé aux personnes majeures." : null;
  const valid = name.trim().length > 0 && iso !== null && !dateError && gender !== "";

  return (
    <>
      <ScreenHeader eyebrow="Étape 1 sur 5" subtitle="Ton prénom et ton âge seront visibles sur ton profil." title="Faisons connaissance" />
      <Input
        autoCapitalize="words"
        autoComplete="given-name"
        label="Prénom"
        maxLength={40}
        onChangeText={setName}
        testID="onboarding-name"
        value={name}
      />
      <BirthdateField day={birth.day} error={dateError} month={birth.month} onChange={setBirth} year={birth.year} />
      <GenderField onChange={setGender} value={gender} />
      {update.isError ? <Text tone="danger">{errorMessage(update.error)}</Text> : null}
      <Nav
        disabled={!valid}
        loading={update.isPending}
        onNext={() => update.mutate({ firstName: name.trim(), birthdate: iso ?? "", gender: gender as Gender }, { onSuccess: onDone })}
      />
    </>
  );
}

function LookingStep({ onBack, onDone, profile }: StepProps) {
  const update = useUpdatePreferences();
  const [audience, setAudience] = useState<Gender[]>(profile.preferences.interestedIn);
  const [range, setRange] = useState({
    minAge: profile.preferences.minAge,
    maxAge: profile.preferences.maxAge === 99 && profile.age ? Math.min(99, profile.age + 10) : profile.preferences.maxAge,
    distance: profile.preferences.maxDistanceKm
  });
  return (
    <>
      <ScreenHeader eyebrow="Étape 2 sur 5" subtitle="Tu pourras changer ces critères à tout moment." title="Qui veux-tu rencontrer ?" />
      <AudienceField onChange={setAudience} value={audience} />
      <RangeFields distance={range.distance} maxAge={range.maxAge} minAge={range.minAge} onChange={setRange} />
      {update.isError ? <Text tone="danger">{errorMessage(update.error)}</Text> : null}
      <Nav
        disabled={audience.length === 0}
        loading={update.isPending}
        onBack={onBack}
        onNext={() =>
          update.mutate(
            { interestedIn: audience, minAge: range.minAge, maxAge: range.maxAge, maxDistanceKm: range.distance },
            { onSuccess: onDone }
          )
        }
      />
    </>
  );
}

function PhotosStep({ onBack, onDone, profile }: StepProps) {
  return (
    <>
      <ScreenHeader eyebrow="Étape 3 sur 5" subtitle="Au moins une photo où l'on voit bien ton visage. Jusqu'à 6." title="Montre-toi" />
      <PhotoGrid maxPhotos={profile.maxPhotos} photos={profile.photos} />
      <Nav disabled={profile.photos.length === 0} onBack={onBack} onNext={onDone} />
    </>
  );
}

function AboutStep({ onBack, onDone, profile }: StepProps) {
  const update = useUpdateProfile();
  const interests = useInterests();
  const [bio, setBio] = useState(profile.bio);
  const [selected, setSelected] = useState(profile.interests.map((i) => i.slug));
  return (
    <>
      <ScreenHeader eyebrow="Étape 4 sur 5" subtitle="Quelques mots et passions aident à lancer la conversation." title="Parle de toi" />
      <Input
        helperText={`${bio.length}/500`}
        label="Bio"
        maxLength={500}
        multiline
        onChangeText={setBio}
        placeholder="Ce qui te fait vibrer, ce que tu cherches…"
        testID="onboarding-bio"
        value={bio}
      />
      <InterestsField all={interests.data ?? []} max={profile.maxInterests} onChange={setSelected} selected={selected} />
      {update.isError ? <Text tone="danger">{errorMessage(update.error)}</Text> : null}
      <Nav loading={update.isPending} onBack={onBack} onNext={() => update.mutate({ bio, interests: selected }, { onSuccess: onDone })} />
    </>
  );
}

function LocationStep({ onBack, onDone, profile }: StepProps) {
  const update = useUpdateLocation();
  const [error, setError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const done = useMemo(() => profile.hasLocation, [profile.hasLocation]);

  useEffect(() => {
    if (update.isError) {
      setError(errorMessage(update.error));
    }
  }, [update.error, update.isError]);

  const locate = async () => {
    setError(null);
    setLocating(true);
    const result = await readApproximateLocation();
    setLocating(false);
    if ("error" in result) {
      setError(result.error);
      return;
    }
    update.mutate(result);
  };

  return (
    <>
      <ScreenHeader eyebrow="Étape 5 sur 5" subtitle="Pour te proposer des personnes près de toi." title="Où te trouves-tu ?" />
      <Card>
        <Text weight="semibold">Ta vie privée est protégée</Text>
        <Text tone="muted" variant="label">
          Ta position est arrondie à environ 1 km avant d&apos;être enregistrée. Les autres ne voient qu&apos;une distance approximative,
          jamais ton adresse. Tu peux masquer la distance ou effacer ta position dans les réglages.
        </Text>
      </Card>
      {done ? (
        <Card tone="primary">
          <Text weight="semibold">Position enregistrée{profile.city ? ` · ${profile.city}` : ""}</Text>
        </Card>
      ) : null}
      {error ? <Text tone="danger">{error}</Text> : null}
      {done ? (
        <Button label="Commencer à découvrir" onPress={onDone} size="lg" testID="onboarding-finish" />
      ) : (
        <Button
          label="Utiliser ma position"
          loading={locating || update.isPending}
          onPress={() => void locate()}
          size="lg"
          testID="onboarding-locate"
        />
      )}
      <View style={styles.nav}>
        {onBack ? <Button label="Retour" onPress={onBack} style={styles.navBack} variant="secondary" /> : null}
        {!done ? <Button label="Plus tard" onPress={onDone} style={styles.navNext} variant="ghost" /> : null}
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  nav: {
    flexDirection: "row",
    gap: spacing.sm,
    marginTop: spacing.sm
  },
  navBack: {
    flex: 1
  },
  navNext: {
    flex: 2
  },
  progress: {
    flexDirection: "row",
    gap: spacing.xs,
    justifyContent: "center"
  },
  progressDot: {
    width: 28,
    height: 4,
    borderRadius: radii.pill,
    backgroundColor: colors.border
  },
  progressDotActive: {
    backgroundColor: colors.primary
  }
});
