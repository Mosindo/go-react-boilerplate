import React, { useState } from "react";
import { errorMessage } from "../api/client";
import type { Gender, Me, Profile } from "../api/types";
import { GenderChips } from "../components/GenderChips";
import { InterestPicker } from "../components/InterestPicker";
import { LocationCard } from "../components/LocationCard";
import { GENDER_LABEL } from "../domain/labels";
import { profileToInput } from "../domain/profileInput";
import { useMe } from "../hooks/useAuth";
import { useInterests, useSaveProfile } from "../hooks/useProfile";
import type { MainScreenProps } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { ScreenContainer, Section } from "../shared/layout";
import { Button, FormField, Input, Notice } from "../shared/ui";

function EditForm({ profile, onDone }: { profile: Profile; onDone: () => void }) {
  const interests = useInterests();
  const save = useSaveProfile();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [interestIds, setInterestIds] = useState(profile.interests.map((interest) => interest.id));
  const [error, setError] = useState<string | null>(null);

  const submit = () => {
    if (firstName.trim().length === 0) {
      setError("Your first name cannot be empty.");
      return;
    }
    setError(null);
    save.mutate(profileToInput(profile, { firstName: firstName.trim(), gender, bio: bio.trim(), interestIds }), {
      onSuccess: () => {
        showToast("Profile saved.", "success");
        onDone();
      }
    });
  };

  return (
    <ScreenContainer scroll underHeader>
      {error || save.isError ? <Notice kind="error" message={error ?? errorMessage(save.error)} /> : null}
      <Input label="First name" maxLength={50} onChangeText={setFirstName} value={firstName} />
      <FormField label="Gender">
        <GenderChips labels={GENDER_LABEL} onToggle={setGender} selected={[gender]} />
      </FormField>
      <Input
        hint={`${bio.length}/500`}
        label="About you"
        maxLength={500}
        multiline
        onChangeText={setBio}
        style={{ minHeight: 120, textAlignVertical: "top" }}
        value={bio}
      />
      <FormField label="Interests">
        {interests.isPending ? (
          <LoadingView label="Loading interests" />
        ) : interests.isError ? (
          <ErrorView message={errorMessage(interests.error)} onRetry={() => void interests.refetch()} />
        ) : (
          <InterestPicker interests={interests.data} onChange={setInterestIds} selectedIds={interestIds} />
        )}
      </FormField>
      <Button label="Save changes" loading={save.isPending} onPress={submit} />
      <Section title="Location">
        <LocationCard actionLabel="Update my location" currentLabel={profile.locationLabel} />
      </Section>
    </ScreenContainer>
  );
}

export default function EditProfileScreen({ navigation }: MainScreenProps<"EditProfile">) {
  const meQuery = useMe();
  const me: Me | undefined = meQuery.data;
  if (meQuery.isPending) {
    return <LoadingView />;
  }
  if (!me?.profile) {
    return <ErrorView message={errorMessage(meQuery.error)} onRetry={() => void meQuery.refetch()} />;
  }
  return <EditForm onDone={() => navigation.goBack()} profile={me.profile} />;
}
