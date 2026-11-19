import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, LookingFor } from "../../api/types";
import { genderLabels, lookingForLabels } from "../../lib/format";
import { hasLetter } from "../../lib/validation";
import type { AppStackParamList } from "../../navigation/types";
import { useSession, useSetProfile } from "../../state/session";
import { spacing } from "../../theme/tokens";
import { Button, Header, LoadingState, Screen, SectionTitle, Text, TextField, useFeedback } from "../../ui";
import { ChoiceList } from "./ChoiceList";
import { InterestPicker } from "./InterestPicker";

type Props = NativeStackScreenProps<AppStackParamList, "EditProfile">;

export function EditProfileScreen({ navigation }: Props) {
  const { profile } = useSession();
  if (!profile) return <LoadingState />;
  return <EditProfileForm navigation={navigation} />;
}

function EditProfileForm({ navigation }: Pick<Props, "navigation">) {
  const { profile } = useSession();
  const setProfile = useSetProfile();
  const { toast } = useFeedback();
  const p = profile!;
  const [firstName, setFirstName] = useState(p.firstName);
  const [gender, setGender] = useState<Gender>(p.gender);
  const [lookingFor, setLookingFor] = useState<LookingFor>(p.lookingFor);
  const [bio, setBio] = useState(p.bio);
  const [jobTitle, setJobTitle] = useState(p.jobTitle);
  const [city, setCity] = useState(p.city);
  const [interestIds, setInterestIds] = useState(p.interests.map((i) => i.id));

  const save = useMutation({
    mutationFn: () =>
      profileApi.save({ firstName: firstName.trim(), birthdate: p.birthdate, gender, lookingFor, bio, jobTitle, city, interestIds }),
    onSuccess: (next) => {
      setProfile(next);
      toast("Profil mis à jour", "success");
      navigation.goBack();
    }
  });

  return (
    <Screen
      scroll
      footer={
        <View style={styles.footer}>
          {save.error ? <Text tone="danger">{errorMessage(save.error)}</Text> : null}
          <Button
            label="Enregistrer"
            onPress={() => save.mutate()}
            loading={save.isPending}
            disabled={!hasLetter(firstName)}
            testID="edit-save"
          />
        </View>
      }
    >
      <Header title="Modifier mon profil" onBack={navigation.goBack} />
      <View style={styles.form}>
        <TextField
          label="Prénom"
          value={firstName}
          onChangeText={setFirstName}
          maxLength={40}
          error={hasLetter(firstName) ? null : "Indiquez votre prénom."}
        />
        <TextField label="Bio" value={bio} onChangeText={setBio} multiline maxLength={500} counter testID="edit-bio" />
        <TextField label="Métier" value={jobTitle} onChangeText={setJobTitle} maxLength={60} />
        <TextField label="Ville affichée" value={city} onChangeText={setCity} maxLength={80} />
        <SectionTitle>Genre</SectionTitle>
        <ChoiceList
          options={(Object.keys(genderLabels) as Gender[]).map((g) => ({ value: g, label: genderLabels[g] }))}
          selected={[gender]}
          onToggle={setGender}
        />
        <SectionTitle>Je cherche</SectionTitle>
        <ChoiceList
          options={(Object.keys(lookingForLabels) as LookingFor[]).map((l) => ({ value: l, label: lookingForLabels[l] }))}
          selected={[lookingFor]}
          onToggle={setLookingFor}
        />
        <SectionTitle>Centres d’intérêt</SectionTitle>
        <InterestPicker selected={interestIds} onChange={setInterestIds} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md },
  footer: { gap: spacing.xs, paddingVertical: spacing.xs }
});
