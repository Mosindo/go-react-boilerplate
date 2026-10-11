import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage, profileApi, queryKeys } from '../../api';
import type { Preferences } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import type { RootScreenProps } from '../../navigation/types';
import { Button, ErrorState, Loading, Screen, Text, useToast } from '../../ui';
import { PreferencesFields } from './components/PreferencesFields';
import { usePreferences } from './hooks';

export function PreferencesScreen({ navigation }: RootScreenProps<'Preferences'>) {
  const query = usePreferences();
  if (query.isPending) return <Loading />;
  if (query.isError) {
    return (
      <Screen edges={['bottom']}>
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  return <PreferencesForm initial={query.data} onDone={() => navigation.goBack()} />;
}

function PreferencesForm({ initial, onDone }: { initial: Preferences; onDone: () => void }) {
  const qc = useQueryClient();
  const toast = useToast();
  const onError = useErrorToast();
  const [value, setValue] = useState(initial);
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: () => profileApi.savePreferences(value),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.preferences });
      await qc.invalidateQueries({ queryKey: queryKeys.ownProfile });
      toast.success(fr.preferences.saved);
      onDone();
    },
    onError,
  });

  const submit = () => {
    if (value.interestedIn.length === 0) {
      setError(fr.validation.interestedIn);
      return;
    }
    save.mutate();
  };

  return (
    <Screen scroll edges={['bottom']}>
      <Text color="textMuted">{fr.preferences.intro}</Text>
      <PreferencesFields
        value={value}
        onChange={(next) => {
          setValue(next);
          setError(null);
        }}
        interestedInError={error}
      />
      <Button label={fr.common.save} onPress={submit} loading={save.isPending} />
    </Screen>
  );
}
