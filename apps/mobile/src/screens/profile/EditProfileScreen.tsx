import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { saveProfile } from "../../api/profile";
import type { Gender, OwnProfile } from "../../api/types";
import { BioField, Button, ErrorState, GenderPicker, InterestPicker, LoadingState, Screen, Text, TextField, useToast } from "../../components";
import { useInterests, useOwnProfile } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";
import { profileToInput } from "../../lib/profileInput";
import { validateBio, validateFirstName } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

function Form({ profile, onDone }: { profile: OwnProfile; onDone: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [city, setCity] = useState(profile.city);
  const [interestIds, setInterestIds] = useState(profile.interests.map((i) => i.id));
  const [errors, setErrors] = useState<{ firstName?: string | null; bio?: string | null }>({});

  const mutation = useMutation({
    mutationFn: () =>
      saveProfile(profileToInput(profile, { firstName: firstName.trim(), gender, bio: bio.trim(), city: city.trim() || undefined, interestIds })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      toast.show("Profile saved", "success");
      onDone();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const submit = (): void => {
    const next = { firstName: validateFirstName(firstName), bio: validateBio(bio) };
    setErrors(next);
    if (!next.firstName && !next.bio) mutation.mutate();
  };

  return (
    <Screen scroll testID="edit-profile-screen">
      <View style={{ gap: spacing.lg }}>
        <TextField label="First name" value={firstName} onChangeText={setFirstName} error={errors.firstName} testID="profile-name-input" />
        <View style={{ gap: 4 }}>
          <Text variant="label">Birth date</Text>
          <Text tone="muted">{profile.birthDate} (cannot be changed)</Text>
        </View>
        <GenderPicker label="I am" value={gender} onChange={setGender} testIDPrefix="profile-gender" />
        <TextField label="City" value={city} onChangeText={setCity} />
        <BioField value={bio} onChange={setBio} error={errors.bio} />
        <InterestPicker interests={interests.data} loading={interests.isPending} selectedIds={interestIds} onChange={setInterestIds} />
        <Button title="Save" onPress={submit} loading={mutation.isPending} testID="profile-save" />
      </View>
    </Screen>
  );
}

export function EditProfileScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "EditProfile">) {
  const query = useOwnProfile();
  if (query.isPending) return <Screen><LoadingState /></Screen>;
  if (query.isError || !query.data) return <Screen><ErrorState message={errorMessage(query.error, "Profile not found.")} onRetry={() => void query.refetch()} /></Screen>;
  return <Form profile={query.data} onDone={() => navigation.goBack()} />;
}
