import React, { useState } from "react";
import type { MyProfile } from "../api/profile";
import { useMyProfile, useSaveProfile } from "../hooks/useProfileData";
import { messageFromError } from "../lib/errors";
import { isProfileDirty, profileToFormValues, profileToPrivacy, toProfileInput } from "../lib/profileForm";
import { hasErrors, validateProfileForm, type ProfileFormErrors, type ProfileFormValues } from "../lib/validation";
import { showToast } from "../shared/feedback/toast";
import { AboutFields, BasicsFields } from "../shared/forms/ProfileFields";
import { Button } from "../shared/ui/Button";
import { Notice } from "../shared/ui/Notice";
import { ProfileQueryGate } from "./ProfileQueryGate";
import { ProfileSubScreen } from "./ProfileSubScreen";

type Props = { onBack: () => void };

export function ProfileEditScreen({ onBack }: Props) {
  const query = useMyProfile();
  return (
    <ProfileQueryGate onBack={onBack} query={query} title="Edit profile">
      {(profile) => (profile ? <EditForm onBack={onBack} profile={profile} /> : null)}
    </ProfileQueryGate>
  );
}

function EditForm({ onBack, profile }: { onBack: () => void; profile: MyProfile }) {
  const save = useSaveProfile();
  const [initial] = useState(() => profileToFormValues(profile));
  const [form, setForm] = useState<ProfileFormValues>(initial);
  const [errors, setErrors] = useState<ProfileFormErrors>({});
  const [serverError, setServerError] = useState<string | null>(null);
  const dirty = isProfileDirty(form, initial);

  async function submit() {
    const nextErrors = validateProfileForm(form);
    setErrors(nextErrors);
    setServerError(null);
    if (hasErrors(nextErrors)) {
      return;
    }
    try {
      await save.mutateAsync(toProfileInput(form, profileToPrivacy(profile)));
      showToast("Profile saved");
      onBack();
    } catch (error) {
      setServerError(messageFromError(error));
    }
  }

  return (
    <ProfileSubScreen onBack={onBack} testID="profile-edit-screen" title="Edit profile">
      <BasicsFields errors={errors} onChange={setForm} values={form} />
      <AboutFields errors={errors} onChange={setForm} values={form} />
      {serverError ? <Notice title={serverError} tone="danger" /> : null}
      <Button
        disabled={!dirty}
        label="Save changes"
        loading={save.isPending}
        onPress={() => void submit()}
        size="lg"
        testID="profile-save-button"
      />
    </ProfileSubScreen>
  );
}
