import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage } from '../../api/errors';
import { profileApi, profileToInput } from '../../api/profile';
import { GENDERS, type Gender, type MyProfile } from '../../api/types';
import { fr } from '../../i18n/fr';
import type { MainStackParamList } from '../../navigation/types';
import { qk } from '../../query/keys';
import { useMyProfile } from '../../query/hooks';
import { InterestPicker } from '../../shared/forms/InterestPicker';
import { LocationPicker, type LocationValue } from '../../shared/forms/LocationPicker';
import {
  Button,
  ChoiceGroup,
  ErrorState,
  LoadingView,
  Screen,
  Text,
  TextField,
} from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { bumpDiscoverEpoch } from '../../store/appState';
import { spacing } from '../../theme';
import { validateBio, validateFirstName } from '../../validators';

const genderChoices = GENDERS.map((g) => ({ value: g, label: fr.gender[g] }));

export function EditProfileScreen(
  props: NativeStackScreenProps<MainStackParamList, 'EditProfile'>,
) {
  const { data, isPending, isError, error, refetch } = useMyProfile();
  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;
  return <EditProfileForm profile={data} onDone={() => props.navigation.goBack()} />;
}

function EditProfileForm({ profile, onDone }: { profile: MyProfile; onDone: () => void }) {
  const qc = useQueryClient();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [location, setLocation] = useState<LocationValue>({
    city: profile.city,
    latitude: profile.latitude,
    longitude: profile.longitude,
  });
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});

  const save = useMutation({
    mutationFn: () =>
      profileApi.saveProfile(
        profileToInput(profile, {
          firstName: firstName.trim(),
          gender,
          bio: bio.trim(),
          interests,
          city: location.city.trim(),
          latitude: location.latitude ?? profile.latitude,
          longitude: location.longitude ?? profile.longitude,
        }),
      ),
    onSuccess: (saved) => {
      qc.setQueryData(qk.myProfile, saved);
      bumpDiscoverEpoch();
      toast.success(fr.common.saved);
      onDone();
    },
  });

  const submit = () => {
    const next = {
      firstName: validateFirstName(firstName) ?? undefined,
      bio: validateBio(bio) ?? undefined,
      city: location.city.trim() ? undefined : fr.validation.cityRequired,
    };
    setErrors(next);
    if (!next.firstName && !next.bio && !next.city) save.mutate();
  };

  return (
    <Screen
      scroll
      testID="edit-profile-screen"
      footer={
        <Button
          label={fr.common.save}
          onPress={submit}
          loading={save.isPending}
          testID="edit-profile-save"
        />
      }
    >
      <TextField
        testID="edit-firstName"
        label={fr.onboarding.firstName}
        value={firstName}
        onChangeText={setFirstName}
        error={errors.firstName}
        maxLength={40}
        autoCapitalize="words"
      />
      <TextField
        label={fr.profile.birthDateLocked}
        value={`${profile.birthDate} (${fr.common.years(profile.age)})`}
        editable={false}
      />
      <Text variant="label" tone="muted" style={styles.label}>
        {fr.onboarding.gender}
      </Text>
      <ChoiceGroup
        mode="single"
        choices={genderChoices}
        selected={[gender]}
        onToggle={setGender}
        testIDPrefix="edit-gender"
        label={fr.onboarding.gender}
      />
      <TextField
        testID="edit-bio"
        label={fr.interests.bio}
        placeholder={fr.interests.bioPlaceholder}
        value={bio}
        onChangeText={setBio}
        error={errors.bio}
        multiline
        counter={500}
      />
      <Text variant="heading" style={styles.label}>
        {fr.interests.title}
      </Text>
      <InterestPicker selected={interests} onChange={setInterests} />
      <View style={styles.location}>
        <Text variant="heading" style={styles.label}>
          {fr.profile.updateLocation}
        </Text>
        <LocationPicker value={location} onChange={setLocation} cityError={errors.city} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  label: { marginBottom: spacing.sm, marginTop: spacing.sm },
  location: { marginTop: spacing.xl },
});
