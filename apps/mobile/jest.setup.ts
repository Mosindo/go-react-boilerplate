jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn().mockResolvedValue(null),
  setItemAsync: jest.fn().mockResolvedValue(undefined),
  deleteItemAsync: jest.fn().mockResolvedValue(undefined)
}));

jest.mock("expo-location", () => ({
  Accuracy: { Balanced: 3 },
  requestForegroundPermissionsAsync: jest.fn().mockResolvedValue({ status: "denied" }),
  getCurrentPositionAsync: jest.fn(),
  reverseGeocodeAsync: jest.fn().mockResolvedValue([])
}));

jest.mock("expo-image-picker", () => ({
  requestMediaLibraryPermissionsAsync: jest.fn().mockResolvedValue({ granted: false }),
  launchImageLibraryAsync: jest.fn().mockResolvedValue({ canceled: true, assets: [] })
}));

jest.mock("expo-image-manipulator", () => ({
  manipulateAsync: jest.fn(),
  SaveFormat: { JPEG: "jpeg" }
}));

// react-native 0.81's built-in Text mock crashes on its arrow-function component
// (mockComponent reads `.prototype.constructor`); a plain host component is enough here.
jest.mock("react-native/Libraries/Text/Text", () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const React = require("react");
  const Text = React.forwardRef((props: Record<string, unknown>, ref: unknown) =>
    React.createElement("Text", { ...props, ref })
  );
  Text.displayName = "Text";
  return { __esModule: true, default: Text };
});
