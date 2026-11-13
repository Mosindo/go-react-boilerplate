import React from "react";
import type { PublicProfile } from "../../api/types";
import type { RootScreenProps } from "../../navigation/types";
import { useMyProfile } from "../../state/queries";
import { Header, LoadingState, Screen } from "../../ui";
import { ProfileView } from "./ProfileView";

export default function MyPreviewScreen(_: RootScreenProps<"MyPreview">) {
  const { data } = useMyProfile();
  if (!data) {
    return <LoadingState />;
  }
  const preview: PublicProfile = {
    userId: data.userId,
    firstName: data.firstName,
    age: data.age ?? 0,
    gender: data.gender ?? "woman",
    bio: data.bio,
    jobTitle: data.jobTitle,
    heightCm: data.heightCm,
    relationshipGoal: data.relationshipGoal,
    city: data.city,
    distanceKm: null,
    photos: data.photos,
    interests: data.interests,
    sharedInterests: [],
    isDemo: false
  };
  return (
    <Screen padded={false} edges={["top"]}>
      <Header back title="Aperçu de mon profil" />
      <ProfileView profile={preview} />
    </Screen>
  );
}
