import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { saveProfile } from "../api/platform";
import { queryKeys, useInterests, useMyProfile } from "../hooks/queries";
import { errorMessage } from "../i18n/errors";
import { genderShort } from "../i18n/fr";
import { cleanName, isValidFirstName } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text, spacing } from "../shared/ui";

const MAX_INTERESTS = 10;
const BIO_MAX = 500;

export default function EditProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useMyProfile();
  const interestsQuery = useInterests();
  const profile = query.data?.profile;

  const [firstName, setFirstName] = useState("");
  const [bio, setBio] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile && !ready) {
      setFirstName(profile.firstName);
      setBio(profile.bio);
      setInterests(profile.interests.map((i) => i.slug));
      setReady(true);
    }
  }, [profile, ready]);

  if (query.isError) {
    return (
      <ScreenContainer>
        <ErrorView error={query.error} onRetry={() => void query.refetch()} />
      </ScreenContainer>
    );
  }
  if (!profile || !ready) {
    return (
      <ScreenContainer>
        <LoadingView fullScreen />
      </ScreenContainer>
    );
  }

  const toggle = (slug: string) =>
    setInterests((current) =>
      current.includes(slug) ? current.filter((s) => s !== slug) : current.length >= MAX_INTERESTS ? current : [...current, slug]
    );

  const save = async () => {
    setError(null);
    if (!isValidFirstName(firstName)) {
      setError("Indiquez votre prénom (40 caractères maximum).");
      return;
    }
    setBusy(true);
    try {
      const response = await saveProfile({
        firstName: cleanName(firstName),
        birthDate: profile.birthDate,
        gender: profile.gender,
        bio: bio.trim(),
        interests
      });
      queryClient.setQueryData(queryKeys.myProfile, response);
      showSuccess("Profil mis à jour.");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <ScreenContainer keyboardAware scroll testID="edit-profile-screen">
      <Header onBack={() => navigation.goBack()} title="Modifier mon profil" />
      <FormField label="Prénom" maxLength={40} onChangeText={setFirstName} testID="edit-first-name" value={firstName} />
      <View style={{ gap: spacing.xs }}>
        <Text tone="muted" variant="label" weight="medium">
          Présentation
        </Text>
        <Input accessibilityLabel="Présentation" maxLength={BIO_MAX} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
        <Text style={{ textAlign: "right" }} tone="subtle" variant="caption">
          {bio.length}/{BIO_MAX}
        </Text>
      </View>
      <View style={{ gap: spacing.sm }}>
        <Text tone="muted" variant="label" weight="medium">
          Centres d&apos;intérêt ({interests.length}/{MAX_INTERESTS})
        </Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {interestsQuery.data?.map((interest) => (
            <Chip
              disabled={!interests.includes(interest.slug) && interests.length >= MAX_INTERESTS}
              key={interest.slug}
              label={interest.label}
              onPress={() => toggle(interest.slug)}
              selected={interests.includes(interest.slug)}
            />
          ))}
        </View>
      </View>
      <Notice
        message={`${genderShort[profile.gender]} · ${profile.age} ans. Le genre et la date de naissance ne sont pas modifiables.`}
        tone="info"
      />
      {error ? <Notice message={error} tone="danger" /> : null}
      <Button fullWidth label="Enregistrer" loading={busy} onPress={() => void save()} size="lg" testID="edit-save" />
    </ScreenContainer>
  );
}
