import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { saveProfile } from "../api/platform";
import type { OwnProfile } from "../api/types";
import { BirthDateField } from "../components/BirthDateField";
import { InterestPicker } from "../components/InterestPicker";
import { LocationStep } from "../components/LocationStep";
import { PhotoManager } from "../components/PhotoManager";
import { useMyProfile } from "../hooks/queries";
import { splitIsoDate } from "../lib/dates";
import { markDeckStale } from "../lib/deckSignal";
import { errorMessage } from "../lib/errors";
import { GENDER_LABELS } from "../lib/labels";
import { queryKeys } from "../realtime/cache";
import { ErrorView, LoadingView, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Button, Input, Notice, Text } from "../shared/ui";

export default function EditProfileScreen() {
  const { data, isLoading, isError, error, refetch } = useMyProfile();
  if (isLoading) return <LoadingView />;
  if (isError || !data?.profile) return <ErrorView error={error} onRetry={() => void refetch()} />;
  return <Editor profile={data.profile} />;
}

function Editor({ profile }: { profile: OwnProfile }) {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [bio, setBio] = useState(profile.bio);
  const [occupation, setOccupation] = useState(profile.occupation);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Photos are saved immediately by PhotoManager, so the screen follows the live cache for them.
  const { data } = useMyProfile();
  const photos = data?.profile?.photos ?? profile.photos;

  useEffect(() => setError(null), [firstName, bio, occupation, interests]);

  const save = async () => {
    setSaving(true);
    try {
      await saveProfile({
        firstName: firstName.trim(),
        birthDate: profile.birthDate,
        gender: profile.gender,
        bio: bio.trim(),
        occupation: occupation.trim(),
        interests
      });
      await queryClient.invalidateQueries({ queryKey: queryKeys.me });
      void queryClient.invalidateQueries({ queryKey: ["profile"] });
      showSuccess("Profil mis à jour.");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ScreenContainer scroll testID="edit-profile">
      <Header title="Mon profil" onBack={() => navigation.goBack()} />
      <Section title="Photos" description="La première photo est celle qu'on voit en premier.">
        <PhotoManager photos={photos} />
      </Section>
      <Section title="À propos de vous">
        <Input testID="edit-first-name" label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} />
        <BirthDateField value={splitIsoDate(profile.birthDate)} onChange={() => undefined} locked />
        <Text variant="caption" tone="muted">
          Genre : {GENDER_LABELS[profile.gender]}
        </Text>
        <Input testID="edit-bio" label="Bio" value={bio} onChangeText={setBio} multiline maxLength={500} hint={`${bio.length} / 500`} />
        <Input testID="edit-occupation" label="Métier" value={occupation} onChangeText={setOccupation} maxLength={60} />
      </Section>
      <Section title="Centres d'intérêt">
        <InterestPicker selected={interests} onChange={setInterests} />
      </Section>
      <Section title="Localisation" description={profile.city ? `Actuellement : ${profile.city}` : undefined}>
        <LocationStep
          label="Mettre à jour ma position"
          onSaved={() => {
            markDeckStale();
            void queryClient.invalidateQueries({ queryKey: queryKeys.me });
          }}
        />
      </Section>
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="edit-save" label="Enregistrer" onPress={save} loading={saving} />
    </ScreenContainer>
  );
}
