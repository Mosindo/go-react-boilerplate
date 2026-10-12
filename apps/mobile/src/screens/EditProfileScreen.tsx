import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import type { Gender, MyProfile, OwnProfile } from "../api/types";
import { PhotoManager } from "../components/PhotoManager";
import { AboutFields, BasicsFields, type AboutValue, type BasicsValue } from "../components/ProfileForm";
import { useMyProfile, useSaveProfile } from "../hooks/queries";
import { parseBirthDate, splitIsoDate, validateFirstName } from "../lib/validation";
import type { MainStackParamList } from "../navigation/types";
import { spacing } from "../theme";
import { Button, ErrorState, LoadingState, Screen, Text, useToast } from "../ui";

export default function EditProfileScreen(props: NativeStackScreenProps<MainStackParamList, "EditProfile">) {
  const query = useMyProfile();
  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError || !query.data?.profile) {
    return <ErrorState message={errorMessage(query.error, "Profil indisponible.")} onRetry={() => void query.refetch()} />;
  }
  return <Form data={query.data} profile={query.data.profile} onDone={() => props.navigation.goBack()} />;
}

function Form({ data, profile, onDone }: { data: MyProfile; profile: OwnProfile; onDone: () => void }) {
  const toast = useToast();
  const save = useSaveProfile();
  const [basics, setBasics] = useState<BasicsValue>({
    firstName: profile.firstName,
    ...splitIsoDate(profile.birthDate),
    gender: profile.gender as Gender
  });
  const [about, setAbout] = useState<AboutValue>({
    bio: profile.bio,
    occupation: profile.occupation,
    city: profile.city,
    interests: data.interests.map((i) => i.slug)
  });
  const [nameError, setNameError] = useState<string | null>(null);
  const birth = useMemo(() => parseBirthDate(basics), [basics]);

  const submit = async () => {
    const error = validateFirstName(basics.firstName);
    setNameError(error);
    if (error || !birth.iso || !basics.gender) {
      return;
    }
    try {
      await save.mutateAsync({
        firstName: basics.firstName.trim(),
        birthDate: birth.iso,
        gender: basics.gender,
        bio: about.bio.trim(),
        occupation: about.occupation.trim(),
        city: about.city.trim(),
        interests: about.interests
      });
      toast.show("Profil enregistré", "success");
      onDone();
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  return (
    <Screen scroll top={false}>
      <View style={styles.section}>
        <Text variant="heading">Photos</Text>
        <PhotoManager photos={data.photos} />
      </View>
      <View style={styles.section}>
        <Text variant="heading">À propos de vous</Text>
        <BasicsFields value={basics} onChange={setBasics} errors={{ firstName: nameError }} lockBirthDate />
        <AboutFields value={about} onChange={setAbout} />
      </View>
      <Button label="Enregistrer" onPress={() => void submit()} loading={save.isPending} />
    </Screen>
  );
}

const styles = StyleSheet.create({ section: { gap: spacing.lg } });
