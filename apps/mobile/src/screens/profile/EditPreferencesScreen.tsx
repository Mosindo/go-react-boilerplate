import React, { useState } from 'react';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage } from '../../api/errors';
import { profileApi } from '../../api/profile';
import type { Preferences } from '../../api/types';
import { fr } from '../../i18n/fr';
import type { MainStackParamList } from '../../navigation/types';
import { qk } from '../../query/keys';
import { usePreferences } from '../../query/hooks';
import { PreferencesForm } from '../../shared/forms/PreferencesForm';
import { Button, ErrorState, LoadingView, Screen } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { bumpDiscoverEpoch } from '../../store/appState';
import { validateAgeRange } from '../../validators';

export function EditPreferencesScreen({
  navigation,
}: NativeStackScreenProps<MainStackParamList, 'EditPreferences'>) {
  const qc = useQueryClient();
  const { data, isPending, isError, error, refetch } = usePreferences();
  const [draft, setDraft] = useState<Preferences | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const value = draft ?? data;

  const save = useMutation({
    mutationFn: (p: Preferences) => profileApi.savePreferences(p),
    onSuccess: (saved) => {
      qc.setQueryData(qk.preferences, saved);
      bumpDiscoverEpoch();
      toast.success(fr.common.saved);
      navigation.goBack();
    },
  });

  if (isPending) return <LoadingView />;
  if (isError || !value)
    return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  const submit = () => {
    const problem =
      value.interestedIn.length === 0
        ? fr.validation.interestedInRequired
        : validateAgeRange(value.minAge, value.maxAge);
    setFormError(problem);
    if (!problem) save.mutate(value);
  };

  return (
    <Screen
      scroll
      testID="edit-preferences-screen"
      footer={
        <Button
          label={fr.common.save}
          onPress={submit}
          loading={save.isPending}
          testID="prefs-save"
        />
      }
    >
      <PreferencesForm value={value} onChange={setDraft} error={formError} />
    </Screen>
  );
}
