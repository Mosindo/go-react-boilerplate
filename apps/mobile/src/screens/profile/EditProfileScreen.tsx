import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { Gender, MyProfile } from "../../api/types";
import { BirthdateField, GenderField, InterestsField } from "../../components/ProfileFields";
import { useInterests, useMyProfile, useUpdateProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { LoadingView, showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Input, Text } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { ageFromIso, toIsoDate } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "EditProfile">;

export default function EditProfileScreen(props: Props) {
  const profile = useMyProfile();
  if (!profile.data) {
    return <LoadingView />;
  }
  return <EditProfileForm {...props} profile={profile.data} />;
}

function EditProfileForm({ navigation, profile }: Props & { profile: MyProfile }) {
  const update = useUpdateProfile();
  const interests = useInterests();
  const [name, setName] = useState(profile.firstName);
  const [bio, setBio] = useState(profile.bio);
  const [city, setCity] = useState(profile.city);
  const [gender, setGender] = useState<Gender | "">(profile.gender);
  const [selected, setSelected] = useState(profile.interests.map((i) => i.slug));
  const [birth, setBirth] = useState(() => {
    const [year = "", month = "", day = ""] = (profile.birthdate ?? "").split("-");
    return { day, month, year };
  });
  const iso = toIsoDate(birth.day, birth.month, birth.year);
  const dateError = !iso ? "Date invalide." : ageFromIso(iso) < 18 ? "Tu dois avoir au moins 18 ans." : null;

  const save = () =>
    update.mutate(
      { firstName: name.trim(), bio, city: city.trim(), gender: gender as Gender, birthdate: iso ?? undefined, interests: selected },
      {
        onSuccess: () => {
          showToast("Profil enregistré", "success");
          navigation.goBack();
        }
      }
    );

  return (
    <Screen
      footer={
        <Button
          disabled={!name.trim() || !!dateError || !gender}
          label="Enregistrer"
          loading={update.isPending}
          onPress={save}
          testID="edit-save"
        />
      }
      testID="edit-profile"
    >
      <ScreenHeader title="Mon profil" />
      <Input label="Prénom" maxLength={40} onChangeText={setName} value={name} />
      <BirthdateField day={birth.day} error={dateError} month={birth.month} onChange={setBirth} year={birth.year} />
      <GenderField onChange={setGender} value={gender} />
      <Input helperText={`${bio.length}/500`} label="Bio" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
      <Input helperText="Affichée sur ton profil (facultatif)." label="Ville" maxLength={80} onChangeText={setCity} value={city} />
      <InterestsField all={interests.data ?? []} max={profile.maxInterests} onChange={setSelected} selected={selected} />
      {update.isError ? <Text tone="danger">{errorMessage(update.error)}</Text> : null}
    </Screen>
  );
}
