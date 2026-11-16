import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { LookingFor, MyProfile } from "../../api/types";
import { InterestPicker } from "../../components/InterestPicker";
import { OptionList } from "../../components/OptionList";
import { useInterests, useMyProfile, useSaveProfile } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme/tokens";
import { Button, Header, LoadingState, Screen, Text, TextField, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { lookingForLabels } from "../../utils/format";

const lookingOptions = (Object.keys(lookingForLabels) as LookingFor[]).map((l) => ({ value: l, label: lookingForLabels[l] }));

export default function EditProfileScreen({ navigation }: AppScreenProps<"EditProfile">) {
  const me = useMyProfile();
  if (!me.data?.profile) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  return <EditForm profile={me.data.profile} onDone={navigation.goBack} />;
}

function EditForm({ profile, onDone }: { profile: MyProfile; onDone: () => void }) {
  const interests = useInterests();
  const save = useSaveProfile();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [city, setCity] = useState(profile.city);
  const [height, setHeight] = useState(profile.heightCm ? String(profile.heightCm) : "");
  const [lookingFor, setLookingFor] = useState<LookingFor>(profile.lookingFor);
  const [interestIds, setInterestIds] = useState(profile.interests.map((i) => i.id));

  const heightNum = height ? Number(height) : null;
  const heightError = heightNum !== null && (Number.isNaN(heightNum) || heightNum < 120 || heightNum > 230) ? "Entre 120 et 230 cm" : null;

  const submit = async () => {
    try {
      await save.mutateAsync({
        firstName: firstName.trim(),
        birthdate: profile.birthdate,
        gender: profile.gender,
        bio,
        jobTitle,
        city,
        heightCm: heightNum,
        lookingFor,
        interestIds
      });
      showToast("Profil mis à jour", "success");
      onDone();
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <Screen
      scroll
      keyboard
      footer={<Button label="Enregistrer" onPress={submit} loading={save.isPending} disabled={!firstName.trim() || !!heightError} testID="edit-profile-save" />}
    >
      <Header title="Modifier mon profil" onBack={onDone} />
      <View style={styles.form}>
        <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} />
        <TextField label="Bio" value={bio} onChangeText={setBio} multiline maxLength={500} counter testID="edit-bio" />
        <TextField label="Métier" value={jobTitle} onChangeText={setJobTitle} maxLength={60} />
        <TextField label="Ville ou quartier" value={city} onChangeText={setCity} maxLength={80} />
        <TextField
          label="Taille (cm, facultatif)"
          value={height}
          onChangeText={(v) => setHeight(v.replace(/\D/g, "").slice(0, 3))}
          keyboardType="number-pad"
          error={heightError}
        />
        <Text variant="label" tone="muted">
          JE CHERCHE
        </Text>
        <OptionList options={lookingOptions} selected={[lookingFor]} onToggle={setLookingFor} />
        <Text variant="label" tone="muted">
          CENTRES D&apos;INTÉRÊT
        </Text>
        <InterestPicker interests={interests.data} selected={interestIds} onChange={setInterestIds} />
        <Text variant="caption" tone="subtle">
          Ta date de naissance et ton genre ne sont pas modifiables ici pour éviter les abus. Contacte le support en cas d&apos;erreur.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg } });
