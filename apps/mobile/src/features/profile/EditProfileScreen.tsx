import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi, queryKeys, type Gender, type ProfileState } from "../../api";
import { MAX_BIO_LENGTH, MAX_FIRST_NAME_LENGTH } from "../../config";
import { showToast } from "../../shared/feedback";
import { Button, ErrorState, Screen, Skeleton, Text, TextField } from "../../shared/ui";
import { validateBio, validateFirstName } from "../../shared/validation";
import type { MainStackParamList } from "../../navigation/types";
import { GenderPicker } from "./components/GenderPicker";
import { InterestsPicker } from "./components/InterestsPicker";
import { PhotoManager } from "./components/PhotoManager";
import { useProfileState } from "./hooks";
import { profileToInput } from "./profileInput";
import type { MyProfile } from "../../api";

type Props = NativeStackScreenProps<MainStackParamList, "EditProfile">;

export function EditProfileScreen(props: Props) {
  const query = useProfileState();
  const profile = query.data?.profile;
  if (query.isError) {
    return (
      <Screen>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  if (!profile) {
    return (
      <Screen>
        <Skeleton height={200} />
      </Screen>
    );
  }
  return <EditProfileForm {...props} profile={profile} />;
}

function EditProfileForm({ navigation, profile }: Props & { profile: MyProfile }) {
  const queryClient = useQueryClient();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [bio, setBio] = useState(profile.bio);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [interests, setInterests] = useState(() => profile.interests.map((interest) => interest.slug));
  const [errors, setErrors] = useState<{ firstName?: string; bio?: string }>({});

  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (updated) => {
      queryClient.setQueryData<ProfileState>(queryKeys.profile, (state) =>
        state ? { ...state, profile: updated } : state
      );
      showToast("Profil enregistré.");
      navigation.goBack();
    }
  });

  const submit = () => {
    const nextErrors = { firstName: validateFirstName(firstName) ?? undefined, bio: validateBio(bio) ?? undefined };
    setErrors(nextErrors);
    if (nextErrors.firstName || nextErrors.bio) {
      return;
    }
    save.mutate(profileToInput(profile, { firstName: firstName.trim(), bio: bio.trim(), gender, interests }));
  };

  return (
    <Screen
      footer={<Button label="Enregistrer" loading={save.isPending} onPress={submit} testID="edit-profile-save" />}
      scroll
      testID="edit-profile-screen"
    >
      <Text variant="heading">Photos</Text>
      <PhotoManager photos={profile.photos} />
      <TextField
        autoCapitalize="words"
        error={errors.firstName}
        label="Prénom"
        maxLength={MAX_FIRST_NAME_LENGTH}
        onChangeText={setFirstName}
        testID="edit-first-name"
        value={firstName}
      />
      <TextField
        error={errors.bio}
        hint={`${bio.length}/${MAX_BIO_LENGTH}`}
        label="Présentation"
        maxLength={MAX_BIO_LENGTH}
        multiline
        onChangeText={setBio}
        style={{ minHeight: 96, textAlignVertical: "top" }}
        testID="edit-bio"
        value={bio}
      />
      <GenderPicker onChange={setGender} value={gender} />
      <InterestsPicker onChange={setInterests} selected={interests} />
    </Screen>
  );
}
