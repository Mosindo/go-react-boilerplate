import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { updateProfile } from "../../api/dating";
import type { Gender, ProfilePatch } from "../../api/types";
import { ChoiceList } from "../../components/ChoiceList";
import { InterestPicker } from "../../components/InterestPicker";
import { LocationField } from "../../components/LocationField";
import { PhotoGrid } from "../../components/PhotoGrid";
import { useInterests, useMyProfile, useProfileMutation } from "../../hooks/queries";
import { genderLabels } from "../../lib/format";
import { firstNameProblem } from "../../lib/validation";
import type { MainScreenProps } from "../../navigation/types";
import { Button, Header, LoadingState, Screen, Section, Text, TextField, showToast, spacing } from "../../ui";

export default function EditProfileScreen({ navigation }: MainScreenProps<"EditProfile">) {
  const { data: profile } = useMyProfile();
  const interests = useInterests();
  const mutation = useProfileMutation((p: ProfilePatch) => updateProfile(p));
  const [firstName, setFirstName] = useState("");
  const [bio, setBio] = useState("");
  const [jobTitle, setJobTitle] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [slugs, setSlugs] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile) {
      setFirstName(profile.firstName);
      setBio(profile.bio);
      setJobTitle(profile.jobTitle);
      setGender(profile.gender ?? null);
      setSlugs(profile.interests.map((i) => i.slug));
    }
    // Initialise once from the server copy.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profile?.userId]);

  if (!profile) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }

  const save = async () => {
    setError(null);
    const problem = firstNameProblem(firstName);
    if (problem) {
      setError(problem);
      return;
    }
    try {
      await mutation.mutateAsync({ firstName: firstName.trim(), bio, jobTitle, gender: gender ?? undefined, interests: slugs });
      showToast("Profil enregistré", "success");
      navigation.goBack();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  return (
    <Screen
      footer={<Button label="Enregistrer" loading={mutation.isPending} onPress={save} testID="edit-save" />}
      scroll
      testID="edit-profile"
    >
      <Header onBack={navigation.goBack} title="Modifier mon profil" />
      <Section title="Photos">
        <PhotoGrid photos={profile.photos} />
      </Section>
      <Section title="Identité">
        <View style={styles.fields}>
          <TextField autoCapitalize="words" label="Prénom" maxLength={40} onChangeText={setFirstName} value={firstName} />
          <TextField label="Métier (facultatif)" maxLength={80} onChangeText={setJobTitle} value={jobTitle} />
          <TextField label="Bio" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
          <Text tone="subtle" variant="caption">
            Âge : {profile.age ?? "—"} ans. La date de naissance ne peut pas être modifiée.
          </Text>
        </View>
      </Section>
      <Section title="Genre">
        <ChoiceList
          onToggle={setGender}
          options={(Object.keys(genderLabels) as Gender[]).map((g) => ({ value: g, label: genderLabels[g] }))}
          selected={gender ? [gender] : []}
        />
      </Section>
      <Section title="Centres d'intérêt">
        {interests.data ? <InterestPicker catalog={interests.data} onChange={setSlugs} selected={slugs} /> : <LoadingState />}
      </Section>
      <Section title="Position">
        <LocationField city={profile.city} hasLocation={profile.hasLocation} />
      </Section>
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger">
          {error}
        </Text>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  fields: {
    gap: spacing.md
  }
});
