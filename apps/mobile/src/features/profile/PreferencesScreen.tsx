import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../lib/errors";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, ErrorState, Header, InlineError, Loading, Screen, Section, Text, toast } from "../../ui";
import { AgeRangeField, DistanceField, InterestedInPicker } from "./fields";
import { useMyProfile, useUpdatePreferences } from "./hooks";

type Props = NativeStackScreenProps<AppStackParamList, "Preferences">;

export function PreferencesScreen({ navigation }: Props) {
  const me = useMyProfile();
  const update = useUpdatePreferences();
  const [prefs, setPrefs] = useState(me.data?.preferences ?? null);

  if (me.isPending) return <Loading />;
  if (me.isError || !me.data) return <ErrorState message={errorMessage(me.error)} />;
  const value = prefs ?? me.data.preferences;

  const save = async () => {
    try {
      await update.mutateAsync(value);
      toast("Préférences enregistrées", "success");
      navigation.goBack();
    } catch {
      // Error is displayed inline.
    }
  };

  return (
    <Screen
      scroll
      edges={["top", "bottom"]}
      testID="preferences-screen"
      footer={
        <View style={{ gap: spacing.xs }}>
          <InlineError message={update.error ? errorMessage(update.error) : null} />
          <Button label="Enregistrer" loading={update.isPending} onPress={save} testID="preferences-save" />
        </View>
      }
    >
      <Header title="Préférences de rencontre" onBack={() => navigation.goBack()} />
      <Section title="Je souhaite rencontrer">
        <View style={styles.padded}>
          <InterestedInPicker value={value.interestedIn} onChange={(interestedIn) => setPrefs({ ...value, interestedIn })} />
        </View>
      </Section>
      <Section title="Âge">
        <AgeRangeField min={value.minAge} max={value.maxAge} onChange={(minAge, maxAge) => setPrefs({ ...value, minAge, maxAge })} />
      </Section>
      <Section
        title="Distance maximale"
        footer={me.data.profile.hasLocation ? undefined : "Activez votre position dans « Modifier mon profil » pour filtrer par distance."}
      >
        <View style={styles.padded}>
          <DistanceField value={value.maxDistanceKm} onChange={(maxDistanceKm) => setPrefs({ ...value, maxDistanceKm })} />
        </View>
      </Section>
      <Text variant="caption" color="textSubtle">
        Les critères sont réciproques : vous voyez les personnes qui correspondent à vos critères et dont vous correspondez aux leurs.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({ padded: { paddingVertical: spacing.sm } });
