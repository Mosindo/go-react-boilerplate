import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import { errorMessage } from "../../api/client";
import type { Gender, MyProfile, Preferences } from "../../api/types";
import { useMyProfile } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { useAuth } from "../../hooks/useAuth";
import { spacing } from "../../theme/palette";
import { Button, InlineError, ProgressBar, Screen, Text } from "../../ui";
import { AboutForm } from "../profile/components/AboutForm";
import { BasicsForm, type BasicsValue } from "../profile/components/BasicsForm";
import { LocationCard } from "../profile/components/LocationCard";
import { PhotoManager } from "../profile/components/PhotoManager";
import { PreferencesForm } from "../profile/components/PreferencesForm";
import { basicsFromProfile, hasErrors, validateBasics, type BasicsErrors } from "../profile/profileForm";

const STEPS = ["basics", "preferences", "location", "photos", "about"] as const;
type Step = (typeof STEPS)[number];

const TITLES: Record<Step, { title: string; subtitle: string }> = {
  basics: { title: "Faisons connaissance", subtitle: "Ces informations apparaissent sur ton profil." },
  preferences: { title: "Qui aimerais-tu rencontrer ?", subtitle: "Tu pourras changer d'avis à tout moment." },
  location: { title: "Près de chez toi", subtitle: "Pour te proposer des personnes à une distance raisonnable." },
  photos: { title: "Montre-toi", subtitle: "Au moins une photo où l'on voit bien ton visage." },
  about: { title: "Ce qui te ressemble", subtitle: "Une bio et quelques centres d'intérêt facilitent le premier message." }
};

function defaultPreferences(gender: Gender | ""): Preferences {
  const interestedIn: Gender[] = gender === "man" ? ["woman"] : gender === "woman" ? ["man"] : ["woman", "man", "nonbinary"];
  return { interestedIn, minAge: 18, maxAge: 99, maxDistanceKm: 50 };
}

export function OnboardingScreen({ onDone }: { onDone: () => void }) {
  const { signOut } = useAuth();
  const profile = useMyProfile();
  const initial = profile.data;
  const [step, setStep] = useState<Step>(initial?.exists ? (initial.photos.length ? "about" : "photos") : "basics");
  const [basics, setBasics] = useState<BasicsValue>(() => basicsFromProfile(initial));
  const [basicsErrors, setBasicsErrors] = useState<BasicsErrors>({});
  const [prefs, setPrefs] = useState<Preferences>(() =>
    initial?.exists ? initial.preferences : defaultPreferences(initial?.gender ?? "")
  );
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(initial?.interests.map((i) => i.id) ?? []);

  useEffect(() => {
    if (!initial?.exists) {
      setPrefs(defaultPreferences(basics.gender));
    }
  }, [basics.gender, initial?.exists]);

  const apply = (p: MyProfile) => queryClient.setQueryData(keys.myProfile, p);
  const index = STEPS.indexOf(step);
  const next = () => setStep(STEPS[Math.min(index + 1, STEPS.length - 1)]);

  const save = useMutation({
    mutationFn: async (): Promise<MyProfile | null> => {
      switch (step) {
        case "basics": {
          const { errors, birthdate } = validateBasics(basics);
          setBasicsErrors(errors);
          if (hasErrors(errors) || !birthdate) {
            return null;
          }
          return profileApi.save({
            firstName: basics.firstName.trim(),
            birthdate,
            gender: basics.gender as Gender,
            bio,
            city: basics.city.trim()
          });
        }
        case "preferences":
          return profileApi.savePreferences(prefs);
        case "about": {
          const current = queryClient.getQueryData<MyProfile>(keys.myProfile);
          if (!current) return null;
          await profileApi.save({ firstName: current.firstName, birthdate: current.birthdate, gender: current.gender as Gender, bio, city: current.city });
          return profileApi.saveInterests(interestIds);
        }
        default:
          return queryClient.getQueryData<MyProfile>(keys.myProfile) ?? null;
      }
    },
    onSuccess: (result) => {
      if (!result) {
        return;
      }
      apply(result);
      if (step === "about") {
        onDone();
        return;
      }
      next();
    }
  });

  const photos = profile.data?.photos ?? [];
  const canContinue = step === "preferences" ? prefs.interestedIn.length > 0 : step === "photos" ? photos.length > 0 : true;
  const { title, subtitle } = TITLES[step];

  return (
    <Screen scroll keyboard edges={["top", "bottom"]} testID={`onboarding-${step}`}>
      <View style={styles.top}>
        <ProgressBar value={(index + 1) / STEPS.length} />
        <View style={styles.topRow}>
          <Text variant="caption" tone="subtle">
            Étape {index + 1} sur {STEPS.length}
          </Text>
          <Button label="Se déconnecter" variant="ghost" size="sm" onPress={() => void signOut()} />
        </View>
      </View>
      <View style={styles.body}>
        <Text variant="title" accessibilityRole="header">
          {title}
        </Text>
        <Text tone="muted">{subtitle}</Text>
        <View style={styles.content}>
          {step === "basics" ? <BasicsForm value={basics} onChange={setBasics} errors={basicsErrors} /> : null}
          {step === "preferences" ? <PreferencesForm value={prefs} onChange={setPrefs} /> : null}
          {step === "location" ? <LocationCard hasLocation={!!profile.data?.hasLocation} /> : null}
          {step === "photos" ? <PhotoManager photos={photos} /> : null}
          {step === "about" ? <AboutForm bio={bio} interestIds={interestIds} onBioChange={setBio} onInterestsChange={setInterestIds} /> : null}
        </View>
        <InlineError message={save.error ? errorMessage(save.error) : null} />
      </View>
      <View style={styles.footer}>
        {index > 0 ? <Button label="Retour" variant="outline" onPress={() => setStep(STEPS[index - 1])} style={styles.back} /> : null}
        <Button
          label={step === "about" ? "Terminer" : step === "location" && !profile.data?.hasLocation ? "Passer pour l'instant" : "Continuer"}
          size="lg"
          disabled={!canContinue}
          loading={save.isPending}
          onPress={() => save.mutate()}
          style={styles.flex}
          testID="onboarding-next"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: { gap: spacing.xs, paddingTop: spacing.sm },
  topRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  body: { flex: 1, gap: spacing.xs, maxWidth: 560, width: "100%", alignSelf: "center" },
  content: { marginTop: spacing.lg, gap: spacing.md },
  footer: { flexDirection: "row", gap: spacing.sm, marginTop: spacing.xl, maxWidth: 560, width: "100%", alignSelf: "center" },
  back: { minWidth: 110 },
  flex: { flex: 1 }
});
