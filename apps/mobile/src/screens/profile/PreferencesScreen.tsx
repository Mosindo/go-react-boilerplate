import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { updatePreferences, updateProfile } from "../../api/dating";
import type { Gender, ProfilePatch } from "../../api/types";
import { ChoiceList } from "../../components/ChoiceList";
import { Stepper } from "../../components/Stepper";
import { useMyProfile, useProfileMutation } from "../../hooks/queries";
import { interestedInLabels } from "../../lib/format";
import type { MainScreenProps } from "../../navigation/types";
import { Button, Chip, Header, LoadingState, Screen, Section, Text, Toggle, showToast, spacing } from "../../ui";

const DISTANCES = [5, 10, 25, 50, 100, 200, 500];

export default function PreferencesScreen({ navigation }: MainScreenProps<"Preferences">) {
  const { data: profile } = useMyProfile();
  const prefsMutation = useProfileMutation(updatePreferences);
  const privacyMutation = useProfileMutation((p: ProfilePatch) => updateProfile(p));
  const [interestedIn, setInterestedIn] = useState<Gender[]>([]);
  const [minAge, setMinAge] = useState(18);
  const [maxAge, setMaxAge] = useState(99);
  const [distance, setDistance] = useState(50);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile) {
      setInterestedIn(profile.preferences.interestedIn);
      setMinAge(profile.preferences.minAge);
      setMaxAge(profile.preferences.maxAge);
      setDistance(profile.preferences.maxDistanceKm);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profile?.userId]);

  if (!profile) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }

  const save = async () => {
    setError(null);
    if (interestedIn.length === 0) {
      setError("Choisissez au moins une option.");
      return;
    }
    try {
      await prefsMutation.mutateAsync({ interestedIn, minAge, maxAge, maxDistanceKm: distance });
      showToast("Préférences enregistrées", "success");
      navigation.goBack();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const togglePrivacy = async (patch: ProfilePatch) => {
    try {
      await privacyMutation.mutateAsync(patch);
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <Screen
      footer={<Button label="Enregistrer" loading={prefsMutation.isPending} onPress={save} testID="preferences-save" />}
      scroll
      testID="preferences-screen"
    >
      <Header onBack={navigation.goBack} title="Préférences" />
      <Section title="Je souhaite rencontrer">
        <ChoiceList
          multiple
          onToggle={(g) => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]))}
          options={(Object.keys(interestedInLabels) as Gender[]).map((g) => ({ value: g, label: interestedInLabels[g] }))}
          selected={interestedIn}
        />
      </Section>
      <Section
        hint="Les critères s'appliquent dans les deux sens : vous ne voyez que les personnes qui vous recherchent aussi."
        title="Âge"
      >
        <Stepper label="À partir de" max={maxAge} min={18} onChange={setMinAge} suffix=" ans" value={minAge} />
        <Stepper label="Jusqu'à" max={99} min={minAge} onChange={setMaxAge} suffix=" ans" value={maxAge} />
      </Section>
      <Section title="Distance maximale">
        <View style={styles.chips}>
          {DISTANCES.map((d) => (
            <Chip key={d} label={`${d} km`} onPress={() => setDistance(d)} selected={distance === d} testID={`distance-${d}`} />
          ))}
        </View>
      </Section>
      <Section title="Confidentialité">
        <Toggle
          description="Mettez votre profil en pause : vous gardez vos matchs et conversations, mais n'apparaissez plus dans la découverte."
          label="Apparaître dans la découverte"
          onChange={(v) => void togglePrivacy({ discoverable: v })}
          testID="privacy-discoverable"
          value={profile.discoverable}
        />
        <Toggle
          description="Les autres membres ne verront pas à quelle distance vous êtes."
          label="Afficher ma distance"
          onChange={(v) => void togglePrivacy({ showDistance: v })}
          value={profile.showDistance}
        />
      </Section>
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger">
          {error}
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
