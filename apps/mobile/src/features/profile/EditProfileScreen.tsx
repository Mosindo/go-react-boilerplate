import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, MyProfile } from "../../api/types";
import { useInterests, useMyProfile } from "../../hooks";
import type { RootScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, LoadingState, Screen, Text, TextField, showToast } from "../../ui";
import { ageFromIso, isoToFrenchDate, parseFrenchDate } from "../../utils/format";
import { BirthdateField, GenderPicker, InterestsPicker } from "./fields";

export function EditProfileScreen(props: RootScreenProps<"EditProfile">) {
  const profile = useMyProfile();
  if (!profile.data)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  return <EditForm {...props} profile={profile.data} />;
}

function EditForm({ navigation, profile }: RootScreenProps<"EditProfile"> & { profile: MyProfile }) {
  const queryClient = useQueryClient();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [birthdate, setBirthdate] = useState(isoToFrenchDate(profile.birthdate));
  const [gender, setGender] = useState<Gender | null>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [interestIds, setInterestIds] = useState(profile.interests.map((i) => i.id));
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () =>
      profileApi.update({
        firstName: firstName.trim(),
        birthdate: parseFrenchDate(birthdate) ?? undefined,
        gender: gender ?? undefined,
        bio,
        jobTitle,
        interestIds
      }),
    onSuccess: (updated) => {
      queryClient.setQueryData(queryKeys.profile, updated);
      showToast("Profil enregistré.", "success");
      navigation.goBack();
    },
    onError: (e) => setError(errorMessage(e))
  });

  function submit() {
    setError(null);
    const iso = parseFrenchDate(birthdate);
    if (!firstName.trim()) return setError("Indique ton prénom.");
    if (!iso) return setError("Date de naissance invalide (JJ/MM/AAAA).");
    if (ageFromIso(iso) < 18) return setError("Lueur est réservé aux personnes majeures.");
    save.mutate();
  }

  return (
    <Screen
      scroll
      edges={["left", "right", "bottom"]}
      footer={<Button label="Enregistrer" testID="edit-save" loading={save.isPending} onPress={submit} />}
    >
      <View style={styles.form}>
        <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} />
        <BirthdateField value={birthdate} onChange={setBirthdate} />
        <Text variant="heading">Genre</Text>
        <GenderPicker value={gender} onChange={setGender} />
        <TextField label="Bio" testID="edit-bio" value={bio} onChangeText={setBio} multiline maxLength={500} counter={{ max: 500 }} />
        <TextField label="Métier" value={jobTitle} onChangeText={setJobTitle} maxLength={80} />
        <Text variant="heading">Centres d’intérêt</Text>
        {interests.data ? <InterestsPicker all={interests.data} selected={interestIds} onChange={setInterestIds} /> : <LoadingState />}
        {error ? <Text tone="danger">{error}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.lg,
    paddingTop: spacing.md
  }
});
