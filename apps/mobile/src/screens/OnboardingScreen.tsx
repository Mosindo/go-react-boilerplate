import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { savePreferences, saveProfile } from "../api/platform";
import type { Gender, MyProfile, Preferences } from "../api/types";
import { BirthDateField, type BirthDateValue } from "../components/BirthDateField";
import { GenderPicker } from "../components/GenderPicker";
import { InterestPicker } from "../components/InterestPicker";
import { LocationStep } from "../components/LocationStep";
import { PhotoManager } from "../components/PhotoManager";
import { DEFAULT_PREFERENCES, PreferencesForm } from "../components/PreferencesForm";
import { useAuth } from "../hooks/useAuth";
import { useMyProfile } from "../hooks/queries";
import { parseBirthDate, splitIsoDate } from "../lib/dates";
import { errorMessage } from "../lib/errors";
import { firstIncompleteStep, ONBOARDING_STEPS } from "../lib/onboarding";
import { isValidFirstName } from "../lib/validation";
import { queryKeys } from "../realtime/cache";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, FormField, Input, Notice, Text } from "../shared/ui";
import { radii, spacing, useTheme } from "../theme";

const STEPS = ONBOARDING_STEPS;

export default function OnboardingScreen() {
  const { data, isLoading, isError, error, refetch } = useMyProfile();
  const [step, setStep] = useState<number | null>(null);

  useEffect(() => {
    if (data && step === null) setStep(firstIncompleteStep(data.completion));
  }, [data, step]);

  if (isLoading || (data && step === null)) return <LoadingView label="Préparation de votre profil…" />;
  if (!data || step === null) return isError ? <ErrorView error={error} onRetry={() => void refetch()} /> : <LoadingView />;

  return <Wizard my={data} step={step} setStep={setStep} />;
}

function Wizard({ my, step, setStep }: { my: MyProfile; step: number; setStep: (step: number) => void }) {
  const { colors } = useTheme();
  const { logout } = useAuth();
  const queryClient = useQueryClient();
  const next = () => setStep(Math.min(STEPS.length - 1, step + 1));
  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.me });

  return (
    <ScreenContainer scroll testID={`onboarding-step-${step}`}>
      <Header
        title={STEPS[step]}
        subtitle={`Étape ${step + 1} sur ${STEPS.length}`}
        onBack={step > 0 ? () => setStep(step - 1) : undefined}
        right={<Button label="Se déconnecter" variant="ghost" compact onPress={() => void logout()} />}
      />
      <View
        style={[styles.track, { backgroundColor: colors.surfaceMuted }]}
        accessibilityRole="progressbar"
        accessibilityValue={{ min: 0, max: STEPS.length, now: step + 1 }}
      >
        <View style={[styles.bar, { backgroundColor: colors.primary, width: `${((step + 1) / STEPS.length) * 100}%` }]} />
      </View>

      {step === 0 ? (
        <AboutYouStep
          my={my}
          onDone={async () => {
            await refresh();
            next();
          }}
        />
      ) : null}
      {step === 1 ? <PhotosStep my={my} onNext={next} /> : null}
      {step === 2 ? (
        <BioStep
          my={my}
          onDone={async () => {
            await refresh();
            next();
          }}
        />
      ) : null}
      {step === 3 ? (
        <PreferencesStep
          my={my}
          onDone={async () => {
            await refresh();
            next();
          }}
        />
      ) : null}
      {step === 4 ? <LocationStepScreen onDone={refresh} /> : null}
    </ScreenContainer>
  );
}

function AboutYouStep({ my, onDone }: { my: MyProfile; onDone: () => Promise<void> }) {
  const existing = my.profile;
  const [firstName, setFirstName] = useState(existing?.firstName ?? "");
  const [birth, setBirth] = useState<BirthDateValue>(existing ? splitIsoDate(existing.birthDate) : { day: "", month: "", year: "" });
  const [gender, setGender] = useState<Gender | null>(existing?.gender ?? null);
  const [errors, setErrors] = useState<{ name?: string; birth?: string; gender?: string; form?: string }>({});
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    const parsed = parseBirthDate(birth.day, birth.month, birth.year);
    const next = {
      name: isValidFirstName(firstName) ? undefined : "Entrez votre prénom (lettres uniquement).",
      birth: parsed.ok ? undefined : parsed.error,
      gender: gender ? undefined : "Choisissez une option."
    };
    setErrors(next);
    if (next.name || next.birth || next.gender || !parsed.ok || !gender) return;
    setSaving(true);
    try {
      await saveProfile({
        firstName: firstName.trim(),
        birthDate: parsed.iso,
        gender,
        bio: existing?.bio ?? "",
        occupation: existing?.occupation ?? "",
        interests: existing?.interests.map((i) => i.slug) ?? []
      });
      await onDone();
    } catch (e) {
      setErrors({ form: errorMessage(e) });
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <Input
        testID="onb-first-name"
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        error={errors.name}
        autoComplete="given-name"
        maxLength={40}
      />
      <BirthDateField value={birth} onChange={setBirth} error={errors.birth} locked={!!existing} />
      <FormField label="Je suis" error={errors.gender}>
        <GenderPicker value={gender} onChange={setGender} />
      </FormField>
      {errors.form ? <Notice kind="error" message={errors.form} /> : null}
      <Button testID="onb-next" label="Continuer" onPress={submit} loading={saving} />
    </>
  );
}

function PhotosStep({ my, onNext }: { my: MyProfile; onNext: () => void }) {
  const photos = my.profile?.photos ?? [];
  return (
    <>
      <Text tone="muted">
        Ajoutez au moins une photo où l'on vous voit bien. Elles sont compressées et nettoyées de leurs métadonnées (position GPS comprise).
      </Text>
      <PhotoManager photos={photos} />
      <Button testID="onb-next" label="Continuer" onPress={onNext} disabled={photos.length === 0} />
    </>
  );
}

function BioStep({ my, onDone }: { my: MyProfile; onDone: () => Promise<void> }) {
  const profile = my.profile;
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [occupation, setOccupation] = useState(profile?.occupation ?? "");
  const [interests, setInterests] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!profile) return null;
  const submit = async () => {
    setSaving(true);
    setError(null);
    try {
      await saveProfile({
        firstName: profile.firstName,
        birthDate: profile.birthDate,
        gender: profile.gender,
        bio: bio.trim(),
        occupation: occupation.trim(),
        interests
      });
      await onDone();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <Input
        testID="onb-bio"
        label="Quelques mots sur vous"
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
        hint={`${bio.length} / 500`}
        placeholder="Ce qui vous passionne, ce que vous cherchez…"
      />
      <Input testID="onb-occupation" label="Métier (facultatif)" value={occupation} onChangeText={setOccupation} maxLength={60} />
      <FormField label="Centres d'intérêt" hint="Ils aident à trouver des personnes qui vous ressemblent.">
        <InterestPicker selected={interests} onChange={setInterests} />
      </FormField>
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="onb-next" label="Continuer" onPress={submit} loading={saving} />
    </>
  );
}

function PreferencesStep({ my, onDone }: { my: MyProfile; onDone: () => Promise<void> }) {
  const [prefs, setPrefs] = useState<Preferences>(my.preferences ?? DEFAULT_PREFERENCES);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    setSaving(true);
    setError(null);
    try {
      await savePreferences(prefs);
      await onDone();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <PreferencesForm value={prefs} onChange={setPrefs} />
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="onb-next" label="Continuer" onPress={submit} loading={saving} />
    </>
  );
}

function LocationStepScreen({ onDone }: { onDone: () => Promise<unknown> }) {
  return (
    <>
      <Text variant="heading">Où êtes-vous ?</Text>
      <LocationStep onSaved={() => void onDone()} />
    </>
  );
}

const styles = StyleSheet.create({
  track: { height: 6, borderRadius: radii.pill, overflow: "hidden", marginBottom: spacing.sm },
  bar: { height: 6, borderRadius: radii.pill }
});
