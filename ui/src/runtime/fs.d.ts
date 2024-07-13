declare type FSCallback = (line: string) => void;

export declare function setCallback(callback: FSCallback): void;

export declare function resetCallback(): void;
