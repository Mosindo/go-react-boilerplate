import { useEffect, useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api";
import type { Settings } from "../api/types";
import { useToast } from "../components/feedback";
import { PreferencesForm } from "../components/PreferencesForm";
import { Button, ErrorState, Loading, Screen } from "../components/ui";
import { errorMessage } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { keys } from "../state/queries";
import { spacing } from "../theme";
import { View } from "react-native";

type Props = NativeStackScreenProps<RootStackParamList, "Settings">;

export function SettingsScreen({ navigation }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: keys.settings, queryFn: profileApi.getSettings });
  const [value, setValue] = useState<Settings | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (query.data && !value) {
      setValue(query.data);
    }
  }, [query.data, value]);

  if (query.isLoading || (!value && !query.isError)) {
    return <Loading />;
  }
  if (query.isError || !value) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }

  const save = async () => {
    setBusy(true);
    try {
      await profileApi.saveSettings(value);
      await queryClient.invalidateQueries({ queryKey: keys.settings });
      toast.show("Préférences enregistrées", "success");
      navigation.goBack();
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen edges={["bottom"]} scroll>
      <PreferencesForm onChange={setValue} value={value} />
      <View style={{ marginTop: spacing.xl }}>
        <Button loading={busy} onPress={() => void save()} title="Enregistrer" />
      </View>
    </Screen>
  );
}
