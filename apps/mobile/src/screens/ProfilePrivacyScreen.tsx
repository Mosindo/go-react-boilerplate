import React, { useState } from "react";
import type { MyProfile } from "../api/profile";
import { useMyProfile, useSaveProfile } from "../hooks/useProfileData";
import { messageFromError } from "../lib/errors";
import { profileToFormValues, profileToPrivacy, toProfileInput, type PrivacyValues } from "../lib/profileForm";
import { showToast } from "../shared/feedback/toast";
import { Card } from "../shared/ui/Card";
import { Notice } from "../shared/ui/Notice";
import { SwitchRow } from "../shared/ui/SwitchRow";
import { ProfileQueryGate } from "./ProfileQueryGate";
import { ProfileSubScreen } from "./ProfileSubScreen";

export function ProfilePrivacyScreen({ onBack }: { onBack: () => void }) {
  const query = useMyProfile();
  return (
    <ProfileQueryGate onBack={onBack} query={query} title="Privacy">
      {(profile) => (profile ? <PrivacyForm onBack={onBack} profile={profile} /> : null)}
    </ProfileQueryGate>
  );
}

function PrivacyForm({ onBack, profile }: { onBack: () => void; profile: MyProfile }) {
  const save = useSaveProfile();
  const [privacy, setPrivacy] = useState<PrivacyValues>(() => profileToPrivacy(profile));
  const [error, setError] = useState<string | null>(null);

  async function update(patch: Partial<PrivacyValues>, message: string) {
    const previous = privacy;
    const next = { ...privacy, ...patch };
    setPrivacy(next);
    setError(null);
    try {
      await save.mutateAsync(toProfileInput(profileToFormValues(profile), next));
      showToast(message);
    } catch (err) {
      setPrivacy(previous);
      setError(messageFromError(err));
    }
  }

  return (
    <ProfileSubScreen
      onBack={onBack}
      subtitle="You are in control of what people see. Changes apply immediately."
      testID="profile-privacy-screen"
      title="Privacy"
    >
      <Card>
        <SwitchRow
          description="Turn this off to pause your profile. You will not appear in Discover, but your matches and chats stay."
          disabled={save.isPending}
          onValueChange={(value) =>
            void update({ discoverable: value }, value ? "You are visible again" : "Your profile is paused")
          }
          testID="privacy-discoverable"
          title="Show me in Discover"
          value={privacy.discoverable}
        />
        <SwitchRow
          description="People see roughly how far away you are, never your exact location."
          disabled={save.isPending}
          onValueChange={(value) => void update({ showDistance: value }, "Privacy updated")}
          testID="privacy-show-distance"
          title="Show my distance"
          value={privacy.showDistance}
        />
        <SwitchRow
          description="Hide your age on your profile. You must still be 18 or older."
          disabled={save.isPending}
          onValueChange={(value) => void update({ showAge: value }, "Privacy updated")}
          testID="privacy-show-age"
          title="Show my age"
          value={privacy.showAge}
        />
      </Card>
      {error ? <Notice title={error} tone="danger" /> : null}
    </ProfileSubScreen>
  );
}
