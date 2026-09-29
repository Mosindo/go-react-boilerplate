import React, { useState } from "react";
import { DEFAULT_PREFERENCES, type Preferences } from "../api/profile";
import { useMyProfile, usePreferences, useSavePreferences } from "../hooks/useProfileData";
import { messageFromError } from "../lib/errors";
import {
  hasErrors,
  validatePreferences,
  type PreferencesErrors,
  type PreferencesValues
} from "../lib/validation";
import { showToast } from "../shared/feedback/toast";
import { LocationCard } from "../shared/forms/LocationCard";
import { PreferencesFields } from "../shared/forms/PreferencesFields";
import { Section } from "../shared/layout/Section";
import { Button } from "../shared/ui/Button";
import { Notice } from "../shared/ui/Notice";
import { ProfileQueryGate } from "./ProfileQueryGate";
import { ProfileSubScreen } from "./ProfileSubScreen";

export function ProfilePreferencesScreen({ onBack }: { onBack: () => void }) {
  const query = usePreferences();
  return (
    <ProfileQueryGate onBack={onBack} query={query} title="Discovery">
      {(preferences) => (
        <PreferencesForm initial={preferences ?? DEFAULT_PREFERENCES} onBack={onBack} />
      )}
    </ProfileQueryGate>
  );
}

function PreferencesForm({ initial, onBack }: { initial: Preferences; onBack: () => void }) {
  const save = useSavePreferences();
  const profile = useMyProfile();
  const [values, setValues] = useState<PreferencesValues>(() => ({ ...initial }));
  const [errors, setErrors] = useState<PreferencesErrors>({});
  const [serverError, setServerError] = useState<string | null>(null);

  async function submit() {
    const nextErrors = validatePreferences(values);
    setErrors(nextErrors);
    setServerError(null);
    if (hasErrors(nextErrors)) {
      return;
    }
    try {
      await save.mutateAsync(values);
      showToast("Preferences saved");
      onBack();
    } catch (error) {
      setServerError(messageFromError(error));
    }
  }

  return (
    <ProfileSubScreen
      onBack={onBack}
      subtitle="Choose who shows up in Discover."
      testID="profile-preferences-screen"
      title="Discovery"
    >
      <PreferencesFields errors={errors} onChange={setValues} values={values} />
      {serverError ? <Notice title={serverError} tone="danger" /> : null}
      <Button
        label="Save preferences"
        loading={save.isPending}
        onPress={() => void submit()}
        size="lg"
        testID="preferences-save-button"
      />
      <Section subtitle="Update your position after you move or travel." title="Location">
        <LocationCard hasLocation={profile.data?.hasLocation ?? false} />
      </Section>
    </ProfileSubScreen>
  );
}
