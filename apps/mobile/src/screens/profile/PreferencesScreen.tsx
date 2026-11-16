import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Preferences } from "../../api/types";
import { OptionList } from "../../components/OptionList";
import { Stepper } from "../../components/Stepper";
import { useMyProfile, useSavePreferences } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, LoadingState, Screen, Text, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { interestedInLabels } from "../../utils/format";

const options = (Object.keys(interestedInLabels) as Gender[]).map((g) => ({ value: g, label: interestedInLabels[g] }));

export default function PreferencesScreen({ navigation }: AppScreenProps<"Preferences">) {
  const me = useMyProfile();
  if (!me.data?.preferences) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  return <PreferencesForm initial={me.data.preferences} hasLocation={me.data.completeness.hasLocation} onDone={navigation.goBack} />;
}

function PreferencesForm({ initial, hasLocation, onDone }: { initial: Preferences; hasLocation: boolean; onDone: () => void }) {
  const save = useSavePreferences();
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial.interestedIn);
  const [ageMin, setAgeMin] = useState(initial.ageMin);
  const [ageMax, setAgeMax] = useState(initial.ageMax);
  const [distance, setDistance] = useState(initial.maxDistanceKm);

  const toggle = (g: Gender) => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]));
  const submit = async () => {
    try {
      await save.mutateAsync({ interestedIn, ageMin, ageMax, maxDistanceKm: distance });
      showToast("Critères enregistrés", "success");
      onDone();
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <Screen scroll footer={<Button label="Enregistrer" onPress={submit} loading={save.isPending} disabled={!interestedIn.length} testID="preferences-save" />}>
      <Header title="Mes critères" onBack={onDone} />
      <View style={styles.form}>
        <Text variant="label" tone="muted">
          JE SOUHAITE RENCONTRER
        </Text>
        <OptionList options={options} selected={interestedIn} onToggle={toggle} multiple />
        <Text variant="label" tone="muted">
          TRANCHE D&apos;ÂGE
        </Text>
        <Stepper label="Âge minimum" value={ageMin} min={18} max={ageMax} onChange={setAgeMin} testID="age-min" />
        <Stepper label="Âge maximum" value={ageMax} min={ageMin} max={99} onChange={setAgeMax} testID="age-max" />
        <Text variant="label" tone="muted">
          DISTANCE MAXIMALE
        </Text>
        <Stepper
          label="Distance"
          value={distance}
          min={1}
          max={500}
          step={distance < 20 ? 1 : distance < 100 ? 5 : 25}
          unit=" km"
          onChange={setDistance}
          testID="distance"
        />
        {!hasLocation ? (
          <Text variant="caption" tone="subtle">
            Tu n&apos;as pas partagé ta position : la distance n&apos;est pas appliquée. Active-la dans Confidentialité et compte.
          </Text>
        ) : null}
        <Text variant="caption" tone="subtle">
          Les préférences sont réciproques : tu ne verras que des personnes qui souhaitent aussi rencontrer quelqu&apos;un comme toi.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.md } });
