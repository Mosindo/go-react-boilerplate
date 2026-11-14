import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, LookingFor, Profile } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { useMyProfile, useSetMyProfile } from "../../state/queries";
import { spacing } from "../../theme/palette";
import { Button, Chip, Header, InlineError, LoadingState, Screen, Text, TextField, useToast } from "../../ui";
import { genderLabels, genders, lookingForLabels, lookingFors } from "../../utils/labels";
import { isValidFirstName, parseBirthdate } from "../../utils/validation";
import { BirthdateField, splitBirthdate } from "./BirthdateField";
import { InterestPicker } from "./InterestPicker";

type Props = NativeStackScreenProps<MainStackParamList, "EditProfile">;

export function EditProfileScreen(props: Props) {
  const { data } = useMyProfile();
  if (!data?.profile) return <LoadingState />;
  return <EditForm {...props} profile={data.profile} />;
}

function EditForm({ navigation, profile }: Props & { profile: Profile }) {
  const toast = useToast();
  const setMyProfile = useSetMyProfile();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [birth, setBirth] = useState(splitBirthdate(profile.birthdate));
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [lookingFor, setLookingFor] = useState<LookingFor>(profile.lookingFor);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [height, setHeight] = useState(profile.heightCm ? String(profile.heightCm) : "");
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [submitted, setSubmitted] = useState(false);

  const parsedBirth = useMemo(() => parseBirthdate(birth.day, birth.month, birth.year), [birth]);
  const heightCm = height ? Number(height) : null;
  const heightError = heightCm !== null && (!Number.isInteger(heightCm) || heightCm < 100 || heightCm > 250) ? "Entre 100 et 250 cm." : null;
  const nameError = isValidFirstName(firstName) ? null : "Prénom invalide.";

  const save = useMutation({
    mutationFn: () =>
      profileApi.update({
        firstName: firstName.trim(),
        birthdate: "iso" in parsedBirth ? parsedBirth.iso : "",
        gender,
        lookingFor,
        bio,
        jobTitle,
        heightCm,
        interests
      }),
    onSuccess: (next) => {
      setMyProfile(next);
      toast({ title: "Profil enregistré", tone: "success" });
      navigation.goBack();
    }
  });

  const submit = () => {
    setSubmitted(true);
    if (!nameError && !("error" in parsedBirth) && !heightError) save.mutate();
  };

  return (
    <Screen
      footer={
        <View style={styles.footer}>
          <InlineError message={save.isError ? errorMessage(save.error) : null} />
          <Button label="Enregistrer" loading={save.isPending} onPress={submit} size="lg" />
        </View>
      }
      scroll
    >
      <Header onBack={navigation.goBack} title="Modifier mon profil" />
      <View style={styles.form}>
        <TextField error={submitted ? nameError : null} label="Prénom" maxLength={40} onChangeText={setFirstName} value={firstName} />
        <View style={styles.group}>
          <Text tone="muted" variant="label">
            Date de naissance
          </Text>
          <BirthdateField error={submitted && "error" in parsedBirth ? parsedBirth.error : null} onChange={setBirth} value={birth} />
        </View>
        <View style={styles.group}>
          <Text tone="muted" variant="label">
            Genre
          </Text>
          <View style={styles.chips}>
            {genders.map((g) => (
              <Chip key={g} label={genderLabels[g]} onPress={() => setGender(g)} selected={gender === g} />
            ))}
          </View>
        </View>
        <View style={styles.group}>
          <Text tone="muted" variant="label">
            Je cherche
          </Text>
          <View style={styles.chips}>
            {lookingFors.map((l) => (
              <Chip key={l} label={lookingForLabels[l]} onPress={() => setLookingFor(l)} selected={lookingFor === l} />
            ))}
          </View>
        </View>
        <TextField hint={`${bio.length}/500`} label="Bio" maxLength={500} multiline onChangeText={setBio} value={bio} />
        <TextField label="Profession" maxLength={80} onChangeText={setJobTitle} value={jobTitle} />
        <TextField
          error={heightError}
          keyboardType="number-pad"
          label="Taille en cm (facultatif)"
          maxLength={3}
          onChangeText={(t) => setHeight(t.replace(/\D/g, ""))}
          value={height}
        />
        <Text variant="heading">Centres d’intérêt</Text>
        <InterestPicker onChange={setInterests} value={interests} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg, paddingBottom: spacing.lg },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  footer: { gap: spacing.xs }
});
