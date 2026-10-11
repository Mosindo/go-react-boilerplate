import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage, queryKeys, profileApi } from '../../api';
import type { OwnProfile } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { isValidFirstName, splitIsoDate } from '../../lib/validation';
import type { RootScreenProps } from '../../navigation/types';
import { Button, ErrorState, Loading, Screen, useToast } from '../../ui';
import { BioInput, BirthDateInput, FirstNameInput, GenderPicker } from './components/ProfileFields';
import { InterestPicker } from './components/InterestPicker';
import { LocationField } from './components/LocationField';
import { useOwnProfile } from './hooks';

export function EditProfileScreen({ navigation }: RootScreenProps<'EditProfile'>) {
  const profile = useOwnProfile();
  if (profile.isPending) return <Loading />;
  if (profile.isError) {
    return (
      <Screen edges={['bottom']}>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  return <EditForm own={profile.data} onDone={() => navigation.goBack()} />;
}

function EditForm({ own, onDone }: { own: OwnProfile; onDone: () => void }) {
  const qc = useQueryClient();
  const toast = useToast();
  const onError = useErrorToast();
  const [firstName, setFirstName] = useState(own.firstName);
  const [gender, setGender] = useState(own.gender);
  const [city, setCity] = useState(own.city);
  const [bio, setBio] = useState(own.bio);
  const [interestIds, setInterestIds] = useState(own.interests.map((i) => i.id));
  const [submitted, setSubmitted] = useState(false);
  const [day, month, year] = splitIsoDate(own.birthDate);

  const save = useMutation({
    mutationFn: () =>
      profileApi.save({
        firstName: firstName.trim(),
        birthDate: own.birthDate,
        gender,
        bio: bio.trim(),
        city: city.trim(),
        interestIds,
      }),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.ownProfile });
      toast.success(fr.profile.saved);
      onDone();
    },
    onError,
  });

  const nameError = submitted && !isValidFirstName(firstName) ? fr.validation.firstName : null;
  const cityError = submitted && city.trim().length === 0 ? fr.validation.city : null;

  const submit = () => {
    setSubmitted(true);
    if (!isValidFirstName(firstName) || city.trim().length === 0) return;
    save.mutate();
  };

  return (
    <Screen scroll edges={['bottom']}>
      <FirstNameInput value={firstName} onChange={setFirstName} error={nameError} />
      <BirthDateInput value={{ day, month, year }} onChange={() => undefined} disabled />
      <GenderPicker value={gender} onChange={setGender} />
      <LocationField
        city={city}
        onCityChange={setCity}
        cityError={cityError}
        hasLocation={own.hasLocation}
        mode="immediate"
        onSaved={() => void qc.invalidateQueries({ queryKey: queryKeys.ownProfile })}
      />
      <BioInput value={bio} onChange={setBio} />
      <InterestPicker selectedIds={interestIds} onChange={setInterestIds} />
      <Button
        label={fr.common.save}
        onPress={submit}
        loading={save.isPending}
        testID="profile-save"
      />
    </Screen>
  );
}
