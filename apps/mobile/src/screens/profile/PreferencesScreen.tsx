import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useEffect, useState } from "react";
import { View } from "react-native";
import type { Preferences } from "../../api/types";
import { GENDER_OPTIONS, toggleInList } from "../../features/profile/options";
import { usePreferences, useSavePreferences } from "../../hooks/useProfile";
import { strings } from "../../i18n/strings";
import type { ProfileStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView, reportError, showToast } from "../../shared/feedback";
import { ChipGroup, Stepper, SwitchRow } from "../../shared/form";
import { FormScreen } from "../../shared/layout";
import { Button, Card, Text, spacing } from "../../shared/ui";
import {
  firstError,
  MAX_AGE,
  MAX_DISTANCE_MAX_KM,
  MIN_AGE,
  validateAgeRange,
  validateDistance
} from "../../utils/validation";

type Props = NativeStackScreenProps<ProfileStackParamList, "Preferences">;

export function validatePreferences(preferences: Preferences) {
  return firstError(
    preferences.interestedIn.length === 0 ? strings.validation.interestedInRequired : null,
    validateAgeRange(preferences.minAge, preferences.maxAge),
    validateDistance(preferences.maxDistanceKm)
  );
}

export default function PreferencesScreen({ navigation }: Props) {
  const query = usePreferences();
  const save = useSavePreferences();
  const [draft, setDraft] = useState<Preferences | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!draft && query.data) {
      setDraft(query.data);
    }
  }, [draft, query.data]);

  if (query.isError) {
    return (
      <FormScreen edges={["left", "right", "bottom"]}>
        <ErrorView message={strings.errors.generic} onAction={() => void query.refetch()} />
      </FormScreen>
    );
  }
  if (!draft) {
    return (
      <FormScreen edges={["left", "right", "bottom"]}>
        <LoadingView fullScreen />
      </FormScreen>
    );
  }

  const update = (patch: Partial<Preferences>) => setDraft({ ...draft, ...patch });

  async function submit() {
    if (!draft) {
      return;
    }
    const problem = validatePreferences(draft);
    setError(problem);
    if (problem) {
      return;
    }
    try {
      await save.mutateAsync(draft);
      showToast(strings.profile.saved, "success");
      navigation.goBack();
    } catch (saveError) {
      reportError(saveError);
    }
  }

  return (
    <FormScreen
      edges={["left", "right", "bottom"]}
      footer={
        <Button
          label={strings.common.save}
          loading={save.isPending}
          onPress={() => void submit()}
          size="lg"
        />
      }
    >
      <View style={{ gap: spacing.sm }}>
        <Text variant="heading" weight="bold">
          {strings.preferences.whoTitle}
        </Text>
        <ChipGroup
          onToggle={(gender) => update({ interestedIn: toggleInList(draft.interestedIn, gender) })}
          options={GENDER_OPTIONS}
          selected={draft.interestedIn}
        />
      </View>
      <Stepper
        label={strings.onboarding.minAge}
        max={draft.maxAge}
        min={MIN_AGE}
        onChange={(minAge) => update({ minAge })}
        value={draft.minAge}
      />
      <Stepper
        label={strings.onboarding.maxAge}
        max={MAX_AGE}
        min={draft.minAge}
        onChange={(maxAge) => update({ maxAge })}
        value={draft.maxAge}
      />
      <Stepper
        label={strings.onboarding.maxDistance}
        max={MAX_DISTANCE_MAX_KM}
        min={5}
        onChange={(maxDistanceKm) => update({ maxDistanceKm })}
        step={5}
        unit=" km"
        value={Math.max(5, draft.maxDistanceKm)}
      />
      {error ? (
        <Text tone="danger" variant="caption">
          {error}
        </Text>
      ) : null}

      <Text variant="heading" weight="bold">
        {strings.preferences.privacyTitle}
      </Text>
      <Card style={{ gap: spacing.lg }}>
        <SwitchRow
          description={strings.preferences.discoverableHelp}
          onValueChange={(discoverable) => update({ discoverable })}
          title={strings.preferences.discoverable}
          value={draft.discoverable}
        />
        <SwitchRow
          description={strings.preferences.showDistanceHelp}
          onValueChange={(showDistance) => update({ showDistance })}
          title={strings.preferences.showDistance}
          value={draft.showDistance}
        />
      </Card>
    </FormScreen>
  );
}
