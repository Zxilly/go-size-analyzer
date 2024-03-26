import {loadData} from "./utils.ts";
import {useEffect} from "react";

function App() {
  const data = loadData();

  useEffect(() => {
    document.title = data.name
  },[data])

  

  return (
    <>

    </>
  )
}

export default App
