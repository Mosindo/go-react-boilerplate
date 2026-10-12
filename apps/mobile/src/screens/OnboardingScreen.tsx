import { useQueryClient } from "@tanstack/react-query";
import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import type { MyProfile, Preferences } from "../api/types";
import { AboutFields, BasicsFields, type AboutValue, type BasicsValue } from "../components/ProfileForm";
import { PhotoManager } from "../components/PhotoManager";
import { DEFAULT_PREFERENCES, PreferencesFields } from "../components/PreferencesForm";
import { useAuth } from "../hooks/useAuth";
import { keys, useMyProfile, useSavePreferences, useSaveProfile } from "../hooks/queries";
import { useLocationUpdate } from "../hooks/useLocationUpdate";
import { parseBirthDate, splitIsoDate, validateFirstName } from "../lib/validation";
import { APP_NAME, radii, spacing, useTheme } from "../theme";
import { Button, ErrorState, LoadingState, Screen, Text, useToast } from "../ui";

type Step = 0 | 1 | 2 | 3;
const TITLES = ["À propos de vous", "Quelques mots sur vous", "Vos photos", "Vos préférences"];

/** Resume where the user stopped: the API reports what is still missing. */
function firstStep(data: MyProfile): Step {
  if (!data.profile) {
    return 0;
  }
  if (data.missing.includes("photos")) {
    return 2;
  }
  return 3;
}

export default function OnboardingScreen() {
  const query = useMyProfile();
  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError || !query.data) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }
  return <Wizard data={query.data} />;
}

function Wizard({ data }: { data: MyProfile }) {
  const { colors } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const { signOut } = useAuth();
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();
  const location = useLocationUpdate();

  const [step, setStep] = useState<Step>(() => firstStep(data));
  const [basics, setBasics] = useState<BasicsValue>(() => {
    const parts = data.profile ? splitIsoDate(data.profile.birthDate) : { day: "", month: "", year: "" };
    return { firstName: data.profile?.firstName ?? "", ...parts, gender: data.profile?.gender ?? null };
  });
  const [about, setAbout] = useState<AboutValue>({
    bio: data.profile?.bio ?? "",
    occupation: data.profile?.occupation ?? "",
    city: data.profile?.city ?? "",
    interests: data.interests.map((i) => i.slug)
  });
  const [prefs, setPrefs] = useState<Preferences>(data.preferences ?? DEFAULT_PREFERENCES);
  const [errors, setErrors] = useState<{
    firstName?: string | null;
    birthDate?: string | null;
    gender?: string | null;
    prefs?: string | null;
  }>({});
  const [locationAsked, setLocationAsked] = useState(data.profile?.hasLocation ?? false);

  const birth = useMemo(() => parseBirthDate(basics), [basics]);

  const next = async () => {
    try {
      if (step === 0) {
        const e = {
          firstName: validateFirstName(basics.firstName),
          birthDate: birth.error,
          gender: basics.gender ? null : "Choisissez une option."
        };
        setErrors(e);
        if (e.firstName || e.birthDate || e.gender) {
          return;
        }
        setStep(1);
      } else if (step === 1) {
        if (!basics.gender || !birth.iso) {
          setStep(0);
          return;
        }
        await saveProfile.mutateAsync({
          firstName: basics.firstName.trim(),
          birthDate: birth.iso,
          gender: basics.gender,
          bio: about.bio.trim(),
          occupation: about.occupation.trim(),
          city: about.city.trim(),
          interests: about.interests
        });
        setStep(2);
      } else if (step === 2) {
        if (data.photos.length === 0) {
          toast.show("Ajoutez au moins une photo.", "error");
          return;
        }
        setStep(3);
      } else {
        if (prefs.interestedIn.length === 0) {
          setErrors({ prefs: "Choisissez au moins une option." });
          return;
        }
        await savePreferences.mutateAsync(prefs);
        await client.invalidateQueries({ queryKey: keys.me });
      }
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const enableLocation = async () => {
    const result = await location.update();
    setLocationAsked(true);
    if (result.ok) {
      client.setQueryData(keys.me, result.profile);
      toast.show("Position enregistrée (approximative)", "success");
    } else {
      toast.show(
        result.reason === "denied" ? "Localisation refusée : vous pourrez l'activer plus tard." : "Position indisponible pour le moment.",
        "info"
      );
    }
  };

  const saving = saveProfile.isPending || savePreferences.isPending;

  return (
    <Screen scroll>
      <View style={styles.header}>
        <Text variant="label" tone="primary">
          {APP_NAME} · Étape {step + 1} sur 4
        </Text>
        <View style={styles.progress}>
          {[0, 1, 2, 3].map((i) => (
            <View key={i} style={[styles.bar, { backgroundColor: i <= step ? colors.primary : colors.border }]} />
          ))}
        </View>
        <Text variant="title">{TITLES[step]}</Text>
      </View>

      {step === 0 ? <BasicsFields value={basics} onChange={setBasics} errors={errors} lockBirthDate={!!data.profile} /> : null}
      {step === 1 ? <AboutFields value={about} onChange={setAbout} /> : null}
      {step === 2 ? <PhotoManager photos={data.photos} /> : null}
      {step === 3 ? (
        <View style={styles.group}>
          <PreferencesFields value={prefs} onChange={setPrefs} error={errors.prefs} />
          <View style={[styles.locationBox, { backgroundColor: colors.surfaceMuted }]}>
            <Text variant="label">Voir des profils près de chez vous</Text>
            <Text variant="caption" tone="muted">
              Votre position sert uniquement à calculer des distances approximatives. Elle est arrondie à environ 1 km et jamais affichée
              précisément.
            </Text>
            <Button
              label={locationAsked ? "Actualiser ma position" : "Activer la localisation"}
              variant="secondary"
              onPress={() => void enableLocation()}
              loading={location.loading}
            />
          </View>
        </View>
      ) : null}

      <View style={styles.footer}>
        <Button label={step === 3 ? "Terminer" : "Continuer"} onPress={() => void next()} loading={saving} />
        {step > 0 ? (
          <Button label="Retour" variant="ghost" onPress={() => setStep((s) => (s - 1) as Step)} />
        ) : (
          <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { gap: spacing.sm, marginTop: spacing.md },
  progress: { flexDirection: "row", gap: spacing.xs },
  bar: { flex: 1, height: 5, borderRadius: 3 },
  group: { gap: spacing.xl },
  locationBox: { borderRadius: radii.lg, padding: spacing.lg, gap: spacing.sm },
  footer: { gap: spacing.sm, marginTop: spacing.md }
});
