import React from "react";
import { useMyProfile } from "../hooks/useProfileData";
import { PhotoManager } from "../shared/forms/PhotoManager";
import { ProfileQueryGate } from "./ProfileQueryGate";
import { ProfileSubScreen } from "./ProfileSubScreen";

export function ProfilePhotosScreen({ onBack }: { onBack: () => void }) {
  const query = useMyProfile();
  return (
    <ProfileQueryGate onBack={onBack} query={query} title="Your photos">
      {(profile) => (
        <ProfileSubScreen
          onBack={onBack}
          subtitle="Your first photo is the one people see first."
          testID="profile-photos-screen"
          title="Your photos"
        >
          <PhotoManager photos={profile?.photos ?? []} />
        </ProfileSubScreen>
      )}
    </ProfileQueryGate>
  );
}
