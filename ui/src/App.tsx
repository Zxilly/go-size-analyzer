import {loadData} from "./utils.ts";
import {useEffect} from "react";

function App() {
  const data = loadData();

  useEffect(() => {
    console.log(data)
  },[data])
  return (
    <>

    </>
  )
}

export default App
