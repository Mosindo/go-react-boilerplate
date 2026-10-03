export type RootStackParamList = {
  Tabs: undefined;
  Chat: { matchId: string; userId: string; name: string };
  ProfileDetail: { userId: string; name: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Activity: undefined;
  Profile: undefined;
};
