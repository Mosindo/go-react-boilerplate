import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { friendlyError } from "../api/errors";
import { saveProfile } from "../api/platform";
import type { MeProfile } from "../api/types";
import {
  AboutFields,
  IdentityFields,
  LocationPicker,
  SectionCard,
  type AboutState,
  type IdentityState,
  type LocationState
} from "../components/profileFields";
import { queryKeys, useInterests, useMyProfile } from "../hooks/queries";
import { isoToParts, validateFirstName } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Notice, Text, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;

export default function EditProfileScreen(props: Props) {
  const me = useMyProfile();
  const interests = useInterests();
  if (me.isLoading || interests.isLoading) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement…" />
      </SafeAreaLayout>
    );
  }
  if (!me.data?.profile || !interests.data) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger votre profil." onAction={() => void me.refetch()} />
      </SafeAreaLayout>
    );
  }
  return <EditForm {...props} catalogue={interests.data} me={me.data} />;
}

function EditForm({
  navigation,
  me,
  catalogue
}: Props & { me: MeProfile; catalogue: NonNullable<ReturnType<typeof useInterests>["data"]> }) {
  const queryClient = useQueryClient();
  const profile = me.profile!;
  const [identity, setIdentity] = useState<IdentityState>({
    firstName: profile.firstName,
    birthDate: isoToParts(profile.birthDate),
    gender: profile.gender
  });
  const [location, setLocation] = useState<LocationState>({ latitude: profile.latitude, longitude: profile.longitude, city: profile.city });
  const [about, setAbout] = useState<AboutState>({
    bio: profile.bio,
    occupation: profile.occupation,
    lookingFor: profile.lookingFor,
    interests: profile.interests
  });
  const [nameError, setNameError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const save = async () => {
    const invalid = validateFirstName(identity.firstName);
    setNameError(invalid);
    if (invalid || !identity.gender || !location) {
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await saveProfile({
        firstName: identity.firstName.trim(),
        birthDate: profile.birthDate,
        gender: identity.gender,
        bio: about.bio.trim(),
        occupation: about.occupation.trim(),
        lookingFor: about.lookingFor,
        city: location.city,
        latitude: location.latitude,
        longitude: location.longitude,
        interests: about.interests
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.me });
      navigation.goBack();
    } catch (err) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled" testID="screen-edit-profile">
          <Text variant="title" weight="bold">
            Modifier mon profil
          </Text>
          <SectionCard>
            <IdentityFields birthDateLocked errors={{ firstName: nameError }} onChange={setIdentity} value={identity} />
          </SectionCard>
          <SectionCard>
            <AboutFields catalogue={catalogue} onChange={setAbout} value={about} />
          </SectionCard>
          <SectionCard>
            <Text variant="heading" weight="bold">
              Localisation
            </Text>
            <LocationPicker onChange={setLocation} value={location} />
          </SectionCard>
          {error ? <Notice title={error} tone="danger" /> : null}
          <Button label="Enregistrer" loading={saving} onPress={() => void save()} testID="save-profile" />
          <Button label="Annuler" onPress={() => navigation.goBack()} variant="ghost" />
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  scroll: { padding: spacing.xl, gap: spacing.lg, paddingBottom: spacing.xxxl }
});
